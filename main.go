package main

import (
	"context"
	"fmt"
	stdLog "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/javorszky/battlesnek/pkg/web"
)

func main() {
	// UNIX Time is faster and smaller than most timestamps
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if err := run(); err != nil {
		log.Fatal().Msgf("ran into a problem: %s\n", err)
	}
}

func run() error {
	// Make the shutdown channels
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	serverErrors := make(chan error, 1)

	// create an API
	a := web.NewApp(shutdown)

	errorLogger := stdLog.New(
		log.With().Str("component", "http.server").Logger(),
		"--ERR",
		stdLog.LstdFlags,
	)

	// Configure api server with our mux.
	api := http.Server{
		Addr:              "localhost:8000",
		Handler:           a,
		ReadTimeout:       3 * time.Second,
		ReadHeaderTimeout: 100 * time.Millisecond,
		IdleTimeout:       1 * time.Second,
		WriteTimeout:      1 * time.Second,
		ErrorLog:          errorLogger,
	}

	// Start server and listen for server errors.
	go func() {
		log.Info().Msgf("server startup")
		serverErrors <- a.Start()
	}()

	// Block until something happens (server crashes unrecoverably, or we terminate manually).
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		log.Info().Str("status", "shutdown started").Str("signal", sig.String()).Msg("shutdown")
		defer log.Info().Str("status", "shutdown finished").Str("signal", sig.String()).Msg("shutdown")

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Asking listener to shut down and shed load.
		if err := api.Shutdown(ctx); err != nil {
			api.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}
