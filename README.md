# blik

A lightweight static file server with per-directory config, markdown rendering with syntax highlighting, archive browsing, structured data viewers, file-type icons, image thumbnails, and media info pages.

## Features

- **Static file serving** — serves any directory over HTTP with styled directory listings
- **Per-directory config** — drop a `.blik` INI file in any folder to set handlers and options locally; settings cascade from parent directories
- **File-type icons** — directory listings show tabler SVG icons for known file types (from MIT-free-icons)
- **Image thumbnails** — shows small previews for image files; auto-generated via background workers with optional file watcher
- **Markdown rendering** — serves `.md` files as formatted HTML with CSS-class-based syntax highlighting (no Chroma)
- **Archive browsing** — view contents of `.zip`, `.tar`, `.tar.gz` files without extracting
- **Structured data views** — renders JSON, XML, YAML, TOML, and INI files as expandable trees; CSV and TSV as sortable tables
- **Image info pages** — `?info` on image files shows dimensions, format, and file size
- **Media info pages** — `?info` on video/audio files shows format, file size, and an inline player
- **Theme toggle** — light/dark mode with a soothing colour palette; respects `prefers-color-scheme`
- **Listing layouts** — single-column table, dual-column, or triple-column grid; stores preference in localStorage
- **Symlink control** — disallow symlink following per-directory via `symlinks=false`
- **Security headers** — CSP (`default-src 'none'`), HSTS, X-Content-Type-Options, Referrer-Policy
- **Template system** — custom `.gohtml` templates for listing, markdown, archive, data, and CSS
- **Graceful shutdown** — handles SIGINT/SIGTERM via `climate/daemon`
- **Config auto-reload** — `.blik` changes are picked up without restart via fsnotify

## Usage

```
blik [-l host] [-p port] [-d root] [-t templates] [-i icons] [-v]
```

| Flag | Env | Default | Purpose |
|------|-----|---------|---------|
| `-l` | `BLIK_HOST` | `""` | Listen address |
| `-p` | `BLIK_PORT` | `8080` | Listen port |
| `-d` | `BLIK_ROOT` | (required) | Root directory to serve |
| `-t` | `BLIK_TEMPLATES` | (builtin) | Path to `.gohtml` template directory |
| `-i` | `BLIK_ICONS` | `~/src/MIT-free-icons/icons` | Path to SVG icon directory |
| `-v` | — | — | Print version and exit |

Environment variable `DOMAIN` sets the `Server` response header.

### `.blik` config

Place an INI file named `.blik` in any served directory. Section names are comma-separated glob patterns. A special `[blik]` section sets directory-level options. Config cascades from parents — child `.blik` values override parent values, changes are picked up automatically via file watcher.

```ini
[blik]
thumbnails=true
symlinks=false
thumbs=yes
thumbsize=256w
workers=2
layout=single
mdlayout=single

[*.md, *.markdown]
type=markdown
; template=blog/md   ; optional: use blog/md/render.gohtml

[*.zip, *.tar.gz, *.tgz, *.tar]
type=archive
; template=store/archive   ; optional: use store/archive/archive.gohtml

[*.jpg, *.png, *.gif]
type=info

[*.json]
type=json

[*.xml]
type=xml

[*.yaml, *.yml]
type=yaml

[*.toml]
type=toml

[*.ini, *.env]
type=ini

[*.csv]
type=csv

[*.tsv]
type=tsv

[*.mp4, *.webm, *.mp3, *.flac, *.wav]
type=info
```

| Section | Key | Description |
|---------|-----|-------------|
| `[blik]` | `thumbnails` | Set to `false` to hide image `.thumb` files and disable thumbnail display (default `true`) |
| `[blik]` | `symlinks` | Set to `false` to return 404 for symlinks (default `true`) |
| `[blik]` | `index` | Comma-separated list of filenames to serve instead of the directory listing (e.g. `index=index.html,index.md`) |
| `[blik]` | `thumbs` | Set to `yes` to auto-generate JPEG thumbnails on startup (default unset) |
| `[blik]` | `thumbsize` | Thumbnail size: `256w` (width), `128h` (height), `128x128` (crop to square, default `256w`) |
| `[blik]` | `workers` | Number of parallel thumbnail generator workers (default `1`) |
| `[blik]` | `layout` | Listing layout: `single`, `dual`, or `triple` (default `single`) |
| `[blik]` | `mdlayout` | Markdown layout: `single` or `dual` (two-column text, default `single`) |
| `[blik]` | `csp` | Content-Security-Policy header value |
| `[blik]` | `hsts` | Strict-Transport-Security header value |
| `[*.ext]` | `type=markdown` | Render matching files as formatted HTML via Goldmark |
| `[*.ext]` | `type=archive` | Serve matching files as downloads; show archive tree on `?info` |
| `[*.ext]` | `type=info` | Show an info page (image/media dimensions or metadata) on `?info` |
| `[*.ext]` | `type=json` | Render as an expandable JSON tree |
| `[*.ext]` | `type=xml` | Render as an expandable XML tree |
| `[*.ext]` | `type=yaml` | Render as an expandable YAML tree |
| `[*.ext]` | `type=toml` | Render as an expandable TOML tree |
| `[*.ext]` | `type=ini` | Render as an expandable INI tree (also covers `.env` files) |
| `[*.ext]` | `type=csv` | Render as a sortable table |
| `[*.ext]` | `type=tsv` | Render as a sortable table (tab-separated) |

Append `?raw` to any parsed file to serve the raw content unprocessed. Append `?info` to show metadata.

### Icons

Directory listings show inline SVG icons from [MIT-free-icons](https://github.com/WJR1986/MIT-free-icons) (a fork of tabler-icons). Clone the repo to the default location:

```sh
git clone https://github.com/WJR1986/MIT-free-icons.git ~/src/MIT-free-icons
```

Override the path with `-i /path/to/icons` or `BLIK_ICONS`.

### Thumbnails

For image files (`.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`, `.bmp`, `.ico`), if a file named `<image>.thumb` exists beside the image, the listing shows a 32×32 preview in place of the file-type icon. `.thumb` files are hidden from the listing.

Set `thumbs=yes` in `[blik]` to have blik auto-generate JPEG thumbnails (quality 50) at startup using the configured `thumbsize` and `workers`. A file watcher picks up new images after startup. `.ico` files use themselves as thumbnails since they are already small.

## Build

```sh
creo           # or go build -o build/blik .
```

Requires Go 1.26+.
