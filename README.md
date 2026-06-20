# blik

A lightweight, no-dependency web file server with per-directory config, markdown rendering and archive browsing.

## Features

- **Static file serving** — serves any directory over HTTP with nice directory listings
- **Per-directory config** — drop a `.blik` INI file in any folder to set options locally
- **Markdown rendering** — serve `.md` files as formatted HTML (or raw with `Accept: text/markdown`)
- **Archive info** — view contents of `.zip`, `.tar`, `.tar.gz` files without extracting
- **Template system** — custom `.gohtml` templates for listing, markdown and archive pages
- **Graceful shutdown** — handles SIGINT/SIGTERM via `climate/daemon`

## Usage

```
blik [-l host] [-p port] [-d root] [-t templates] [-v]
```

| Flag | Env | Default | Purpose |
|------|-----|---------|---------|
| `-l` | `BLIK_HOST` | `""` | Listen address |
| `-p` | `BLIK_PORT` | `8080` | Listen port |
| `-d` | `BLIK_ROOT` | `"."` | Root directory to serve |
| `-t` | `BLIK_TEMPLATES` | (builtin) | Path to `.gohtml` template directory |
| `-v` | — | — | Print version and exit |

### `.blik` config

Place an INI file named `.blik` in any served directory. Supported keys:

```ini
# Hide files matching glob patterns
hide = *.secret
hide = private/

# Set directory-level options
title = My Cool Section
```

Config cascades from parents — child `.blik` values override parent values.

## Build

```sh
creo           # or go build -o build/blik .
```

Requires Go 1.26+.
