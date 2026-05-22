package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/younkyumjin/mansio/internal/server"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

func main() {
	port := flag.Int("port", 8080, "server port")
	host := flag.String("host", "127.0.0.1", "server host")
	dataDir := flag.String("data-dir", "./data", "data directory for SQLite database")
	flag.Parse()

	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		log.Fatalf("failed to create sub filesystem: %v", err)
	}

	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("failed to create data directory: %v", err)
	}

	srv := server.New(distFS, *dataDir)

	addr := net.JoinHostPort(*host, strconv.Itoa(*port))
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second, // slowloris 방지 (T2.8)
	}

	go func() {
		log.Printf("Mansio listening on http://%s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	srv.Shutdown()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}
	log.Println("server stopped")
}
