package main

import (
	"context"
	"net/http"
	"time"

	"github.com/grimdork/climate/daemon"
	"github.com/grimdork/climate/loglines"
)

func serve(cfg *Config, handler http.Handler) {
	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		loglines.Msg("listening on %s, serving %s", cfg.Addr(), cfg.Root)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			loglines.Err("server error: %s", err)
		}
	}()

	<-daemon.BreakChannel()

	loglines.Msg("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		loglines.Err("shutdown error: %s", err)
	}
	loglines.Msg("stopped")
}
