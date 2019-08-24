package web

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server type wraps a HTTP server
type Server struct {
	s *http.Server
}

// NewServer constructs a web server which can then be invoked via the `Server.Run()` command.
func NewServer(router http.Handler, port int) *Server {
	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	return &Server{
		s: s,
	}
}

// Run will start the server and wait for error or shutdown - this will block the caller. Any error encountered
// during server startup or operation will be returned to the caller here. The server can be exit'ed using a
// SIGINT, SIGTERM or SIGQUIT interrupt
func (s *Server) Run() error {
	tchan := make(chan os.Signal)
	echan := make(chan error)
	signal.Notify(tchan, os.Interrupt, os.Kill, syscall.SIGQUIT)

	go func() {
		log.Printf("starting server: host = %s", s.s.Addr)

		if err := s.s.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("server failed to run or exited abnormally: error = %s", err)
			echan <- err
		}
	}()

	var err error
	select {
	case tsig := <-tchan:
		log.Printf("closing server: signal = %s", tsig)
	case err = <-echan:
		log.Printf("error starring server : error = %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.s.Shutdown(ctx); err != nil {
		log.Printf("server shutdown failed to exit gracefully: error = %s", err)
	}
	return err
}
