package main

import (
    "flag"
    "fmt"
    "github.com/gorilla/handlers"
    "github.com/gorilla/mux"
    "log"
    "net/http"
    "osdata/osvtile/mvt"
    "osdata/osvtile/web"
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

    flag.Parse()

    log.Println("server starting")

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
    h = web.NewRequestHandler(web.NewClacksHandler(h))

    // routes
    r.HandleFunc("/status", web.NewStatusHandler())
    r.HandleFunc("/{name:[A-Za-z0-9_]+}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}/tile.mvt", web.NewMVTRequestHandler(ds))
    r.HandleFunc("/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}/tile.mvt", web.NewMVTRequestHandler(ds))
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
