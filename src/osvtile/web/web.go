package web

import (
    "encoding/json"
    "github.com/gorilla/mux"
    "io"
    "log"
    "net/http"
    "os"
    "osdata/osvtile/container/lru"
    "osdata/osvtile/mbtiles"
    "path/filepath"
    "strconv"
    "strings"
    "time"
)

// StatusResponseWriter holds the status code that the server wrote to the client.
// This allows upstream logging of the response
type StatusResponseWriter struct {
    status int
    writer http.ResponseWriter
}

func (s *StatusResponseWriter) Header() http.Header {
    return s.writer.Header()
}

func (s *StatusResponseWriter) Write(b []byte) (int, error) {
    return s.writer.Write(b)
}

func (s *StatusResponseWriter) WriteHeader(status int) {
    s.status = status
    s.writer.WriteHeader(status)
}

// Error type for all HTTP related functions
type Error struct {
    Code    int    `json:"code"`
    Status  int    `json:"-"`
    Message string `json:"message"`
}

func (e *Error) Error() string {
    return e.Message
}

func NewStatusHandler(metrics *Metrics, cache *lru.LRU) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        w.Header().Add("content-type", "application/json")
        w.WriteHeader(http.StatusOK)

        metrics.rw.RLock()
        defer metrics.rw.RUnlock()

        status := map[string]interface{}{
            "cache": cache.Status(),
            "requests": metrics,
        }

        packet, _ := json.Marshal(status)
        w.Header().Add("content-size", strconv.Itoa(len(packet)))
        _, err := w.Write(packet)

        if err != nil {
            log.Printf("failed to write status to client: error = %s", err)
        }

        return
    }
}

// NewClacksHandler - say no more
func NewClacksHandler(h http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Clacks-Overhead", "GNU Terry Pratchett")
        h.ServeHTTP(w, r)
    })
}

// NewRequestHandler wraps a handler func to provide standard request logging and setup
func NewRequestHandler(m *Metrics, h http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now().UTC().UnixNano()

        sw := &StatusResponseWriter{status: http.StatusOK, writer: w}
        h.ServeHTTP(sw, r)
        delta := time.Now().UTC().UnixNano() - start
        log.Printf(
            "Request handled: addr: %s, method: %s, uri:%s, userAgent: %s, startTime: %d, deltaTime: %d, status:%d",
            r.RemoteAddr, r.Method, r.RequestURI, r.UserAgent(), start/1000, delta/1000, sw.status,
        )

        m.Log(r, sw.status)
    })
}

// FontHandler will serve static PBF font files
func NewFontHandler(path string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        stack := vars["stack"]
        file := vars["file"]

        if !strings.HasSuffix(file, "pbf") {
            w.WriteHeader(http.StatusNotFound)
            return
        }

        stacks := strings.Split(stack, ",")

        fontFile, err := os.Open(filepath.Join(path, stacks[0], file))

        if err != nil {
            log.Printf("failed to open font path: requested font = %s, file = %s, error = %s", stacks[0], file, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        defer func() {
            if err := fontFile.Close(); err != nil {
                log.Printf(
                    "failed to close font file: error = %s, path = %s",
                    err, filepath.Join(path, stacks[0], file),
                )
            }
        }()

        w.WriteHeader(http.StatusOK)
        _, err = io.Copy(w, fontFile)

        if err != nil {
            log.Printf("failed to write font to client: requested font = %s, file = %s, error = %s", stacks[0], file, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
    }
}

func NewRasterDEMRequestHandler(d *mbtiles.MBTiles, cache *lru.LRU) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        x, _ := strconv.Atoi(vars["x"])
        y, _ := strconv.Atoi(vars["y"])
        z, _ := strconv.Atoi(vars["z"])

        var err error
        tile, md5 := cache.Get(r.RequestURI)

        if tile == nil {
            tile, err = d.FetchTile(x, y, z)

            if tile == nil {
                w.WriteHeader(http.StatusNotFound)
                return
            }

            if err != nil {
                log.Printf("failed to fetch tile from datasource: error = %s", err)
                w.WriteHeader(http.StatusInternalServerError)
                return
            }

            md5 = cache.Set(r.RequestURI, tile)
        }

        // tiles are in gzip format already
        w.Header().Set("content-length", strconv.Itoa(len(tile)))
        w.Header().Set("content-type", "image/png")
        w.Header().Set("etag", md5)
        w.WriteHeader(http.StatusOK)
        c, err := w.Write(tile)

        if err != nil {
            log.Printf("failed to write tile data to client: error = %s", err)
        }

        if c != len(tile) {
            log.Printf("failed to write whole tile to client: tile size = %d, written = %d", len(tile), c)
        }
    }
}

func NewMVTRequestHandler(d *mbtiles.MBTiles, cache *lru.LRU) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        x, _ := strconv.Atoi(vars["x"])
        y, _ := strconv.Atoi(vars["y"])
        z, _ := strconv.Atoi(vars["z"])

        var err error
        tile, md5 := cache.Get(r.RequestURI)

        if tile == nil {
            tile, err = d.FetchTile(x, y, z)

            if tile == nil {
                w.WriteHeader(http.StatusNotFound)
                return
            }

            if err != nil {
                log.Printf("failed to fetch tile from datasource: error = %s", err)
                w.WriteHeader(http.StatusInternalServerError)
                return
            }

            md5 = cache.Set(r.RequestURI, tile)
        }

        // tiles are in gzip format already
        w.Header().Set("content-length", strconv.Itoa(len(tile)))
        w.Header().Set("content-encoding", "gzip")
        w.Header().Set("content-type", "application/x-protobuf")
        w.Header().Set("etag", md5)
        w.WriteHeader(http.StatusOK)
        c, err := w.Write(tile)

        if err != nil {
            log.Printf("failed to write tile data to client: error = %s", err)
        }

        if c != len(tile) {
            log.Printf("failed to write whole tile to client: tile size = %d, written = %d", len(tile), c)
        }
    }
}

// NotFounderHandler provides extra logging when no route matches
func NotFounderHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNotFound)
}
