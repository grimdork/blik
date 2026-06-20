package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/grimdork/climate/daemon"
	"github.com/grimdork/climate/fx"
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
		fx.Println("{logstamp} {success}listening{@} on {}, serving {}", cfg.Addr(), cfg.Root)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fx.Fprintln(os.Stderr, "{logstamp} {danger}server error:{@} {}", err)
		}
	}()

	<-daemon.BreakChannel()

	fx.Println("{logstamp} {warning}shutting down...{@}")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fx.Fprintln(os.Stderr, "{logstamp} {danger}shutdown error:{@} {}", err)
	}
	fx.Println("{logstamp} {success}stopped{@}")
}
