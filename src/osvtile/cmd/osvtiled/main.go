package main

import (
    "flag"
    "fmt"
    "github.com/gorilla/handlers"
    "github.com/gorilla/mux"
    "log"
    "net/http"
    "osdata/osvtile/container/lru"
    "osdata/osvtile/mvt"
    "osdata/osvtile/web"
    "regexp"
    "strconv"
    "strings"
)

const (
    kb int64 = 1024
    mb int64 = kb * 1024
    gb int64 = mb * 1024
)

func main() {

    flag.Usage = func() {
        fmt.Println("Usage: osvtiled [OPTIONS]\n\nSimple server to deliver Ordnance Survey Zoom Stack MBtiles")
        fmt.Println()
        flag.PrintDefaults()
        fmt.Println()
    }

    port := flag.Int("port", 8080, "port on which to run server")
    cors := flag.Bool("cors", false, "enable cors handling")
    proxy := flag.Bool("proxy", false, "enable proxy header support (when behind nginx, apache etc)")
    mbtiles := flag.String("mbtiles", ".", "location of the mbtiles package to serve up")
    static := flag.String("static", ".", "directory to the root static web content (index.html, style etc)")
    cacheSize := flag.String("cache", "512m", "cache size: format <INTEGER><k|m|g>, e.g. 1g or 512mb")

    flag.Parse()

    log.Println("server starting")

    // parse the cache size
    re := regexp.MustCompile(`([1-9]\d*)([kmg])`)
    matches := re.FindStringSubmatch(strings.TrimSpace(*cacheSize))

    if matches == nil || len(matches) == 0 {
        log.Fatalf("invalid cacheSize value: value = %s", *cacheSize)
    }

    size, _ := strconv.Atoi(matches[1])
    bytesize := int64(0)

    switch matches[2] {
    case "g":
        bytesize = int64(size) * gb
    case "m":
        bytesize = int64(size) * mb
    default:
        bytesize = int64(size) * kb
    }

    cache := lru.New(bytesize)

    // mvt dataasource
    ds, err := mvt.NewMVT(*mbtiles)

    if err != nil {
        log.Fatalf("failed to load MBTiles package: error = %s", err)
    }

    v, err := ds.Version()

    if err != nil {
        log.Fatalf("failed to load MBTiles version: error = %s", err)
    }

    log.Printf("loaded MBtiles package: version = %v", v)

    metrics := web.NewMetrics()

    r := mux.NewRouter()

    // add in the wrappers
    var h http.Handler

    // default handler is our mux router
    h = r

    if *proxy {
        log.Printf("enabled proxy support")
        h = handlers.ProxyHeaders(h)
    }

    if *cors {
        log.Printf("enabled CORS support")
        h = handlers.CORS(
            handlers.AllowCredentials(),
            handlers.AllowedOrigins([]string{"*"}),
            handlers.AllowedHeaders([]string{
                "Content-Type",
                "Cache-Control",
                "ETag",
                "Expires",
                "Last-Modified",
                "Content-Length",
            }),
            handlers.AllowedMethods([]string{
                "GET", "HEAD", "POST", "DELETE", "PUT",
            }),
            handlers.ExposedHeaders([]string{
                "X-Clacks-Overhead",
                "Cache-Control",
                "ETag",
                "Expires",
                "Last-Modified",
            }),
        )(h)
    }

    // default handlers will be the logger and clacks
    h = web.NewRequestHandler(metrics, web.NewClacksHandler(h))

    // routes
    r.HandleFunc("/status", web.NewStatusHandler(metrics, cache))
    r.HandleFunc("/{name:[A-Za-z0-9_]+}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}/tile.mvt", web.NewMVTRequestHandler(ds, cache))
    r.HandleFunc("/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}/tile.mvt", web.NewMVTRequestHandler(ds, cache))
    r.HandleFunc("/fonts/{stack}/{file}", web.NewFontHandler(fmt.Sprintf("%s/fonts", *static)))
    r.PathPrefix("/").Handler(http.FileServer(http.Dir(*static)))
    r.NotFoundHandler = http.HandlerFunc(web.NotFounderHandler)

    // server instance
    s := web.NewServer(
        h,
        *port,
    )

    if err := s.Run(); err != nil {
        log.Printf("failed to start server: error = %s", err)
    }

    log.Println("server closed")
}
