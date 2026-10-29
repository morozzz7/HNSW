// Command server starts the HNSW coursework HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hnsw-coursework/internal/api"
	"hnsw-coursework/internal/dataset"
	"hnsw-coursework/internal/indexmanager"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Getenv("PORT")); err != nil {
		log.Printf("api: %v", err)
		os.Exit(1)
	}
}

// run starts the HTTP server and blocks until ctx is cancelled or the server
// fails. On cancellation it stops accepting connections and shuts down
// gracefully with a bounded timeout.
func run(ctx context.Context, port string) error {
	if port == "" {
		port = "8080"
	}

	store := dataset.NewStore()
	manager := indexmanager.New(store)
	server := api.NewServer(store, manager)

	httpServer := &http.Server{
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		manager.Close()
		return fmt.Errorf("listen on port %s: %w", port, err)
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("api: listening on http://%s/api/v1", ln.Addr().String())
		if err := httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		manager.Close()
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		log.Printf("api: shutdown signal received")
	}

	// Cancel an in-flight build first so SSE handlers can receive the terminal
	// error event and return before Shutdown waits for them.
	manager.Close()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("api: graceful shutdown failed: %v", err)
		_ = httpServer.Close()
		return fmt.Errorf("shutdown: %w", err)
	}
	log.Printf("api: stopped")
	return nil
}
