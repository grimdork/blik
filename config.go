package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/grimdork/climate/arg"
)

type Config struct {
	Host        string
	Port        string
	Root        string
	TemplateDir string
}

func (c *Config) Addr() string {
	return c.Host + ":" + c.Port
}

func parseConfig() *Config {
	opts := arg.New("blik", "Static file server.")

	opts.SetFlag("General", "v", "version", "Show version and exit.")
	opts.SetOption("General", "l", "host", "Address to listen on.", "", false, arg.VarString, nil)
	opts.SetOption("General", "p", "port", "Port to listen on.", "8080", false, arg.VarString, nil)
	opts.SetOption("General", "d", "root", "Root directory to serve.", "", false, arg.VarString, nil)
	opts.SetOption("General", "t", "templates", "Templates directory.", "", false, arg.VarString, nil)
	opts.SetDefaultHelp(true)

	for _, a := range os.Args[1:] {
		if a == "--version" || a == "-v" {
			fmt.Printf("blik %s (commit %s, %s)\n", version, commit, date)
			os.Exit(0)
		}
	}

	if err := opts.Parse(os.Args[1:]); err != nil {
		if !errors.Is(err, arg.ErrNonFatal) {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			opts.PrintHelp()
			os.Exit(2)
		}
	}

	cfg := &Config{
		Host:        opts.GetString("host"),
		Port:        opts.GetString("port"),
		Root:        opts.GetString("root"),
		TemplateDir: opts.GetString("templates"),
	}

	if cfg.Host == "" {
		if v := os.Getenv("BLIK_HOST"); v != "" {
			cfg.Host = v
		}
	}
	if cfg.Port == "8080" {
		if v := os.Getenv("BLIK_PORT"); v != "" {
			cfg.Port = v
		}
	}
	if cfg.Root == "" {
		if v := os.Getenv("BLIK_ROOT"); v != "" {
			cfg.Root = v
		}
	}
	if cfg.TemplateDir == "" {
		if v := os.Getenv("BLIK_TEMPLATES"); v != "" {
			cfg.TemplateDir = v
		}
	}

	if cfg.Root == "" {
		fmt.Fprintf(os.Stderr, "Error: root directory is required (-d/--root or BLIK_ROOT)\n")
		opts.PrintHelp()
		os.Exit(2)
	}

	return cfg
}
