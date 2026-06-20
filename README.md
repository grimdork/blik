# blik

A lightweight static file server with per-directory config, markdown rendering with syntax highlighting, archive browsing, file-type icons, and image thumbnails.

## Features

- **Static file serving** — serves any directory over HTTP with styled directory listings
- **Per-directory config** — drop a `.blik` INI file in any folder to set handlers and options locally; settings cascade from parent directories
- **File-type icons** — directory listings show tabler SVG icons for known file types (from MIT-free-icons)
- **Image thumbnails** — shows small previews for image files with a `.thumb` sidecar file
- **Markdown rendering** — serves `.md` files as formatted HTML with CSS-class-based syntax highlighting (no Chroma)
- **Archive browsing** — view contents of `.zip`, `.tar`, `.tar.gz` files without extracting
- **Image info pages** — `?info` on image files shows dimensions, format, and file size
- **Theme toggle** — light/dark mode with a soothing colour palette; respects `prefers-color-scheme`
- **Symlink control** — disallow symlink following per-directory via `symlinks=false`
- **Template system** — custom `.gohtml` templates for listing, markdown, archive, and CSS
- **Graceful shutdown** — handles SIGINT/SIGTERM via `climate/daemon`

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

### `.blik` config

Place an INI file named `.blik` in any served directory. Section names are comma-separated glob patterns. A special `[blik]` section sets directory-level options. Config cascades from parents — child `.blik` values override parent values.

```ini
[blik]
thumbnails=false
symlinks=false

[*.md, *.markdown]
type=markdown
; template=blog/md   ; optional: use blog/md/render.gohtml

[*.zip, *.tar.gz, *.tgz, *.tar]
type=archive
; template=store/archive   ; optional: use store/archive/archive.gohtml

[*.jpg, *.png]
type=info
```

| Section | Key | Description |
|---------|-----|-------------|
| `[blik]` | `thumbnails` | Set to `false` to hide image `.thumb` files and disable thumbnail display (default `true`) |
| `[blik]` | `symlinks` | Set to `false` to return 404 for symlinks, preventing traversal outside the served root (default `true`) |
| `[blik]` | `index` | Comma-separated list of filenames to serve instead of the directory listing (e.g. `index=index.html,index.md`) |
| `[*.ext]` | `type=markdown` | Render matching files as formatted HTML via Goldmark |
| `[*.ext]` | `type=archive` | Serve matching files as downloads (`application/octet-stream`) and show info page on `?info` |
| `[*.ext]` | `type=info` | Show an info page when the file is accessed with `?info` |

### Icons

Directory listings show inline SVG icons from [MIT-free-icons](https://github.com/WJR1986/MIT-free-icons) (a fork of tabler-icons). Clone the repo to the default location:

```sh
git clone https://github.com/WJR1986/MIT-free-icons.git ~/src/MIT-free-icons
```

Override the path with `-i /path/to/icons` or `BLIK_ICONS`.

### Thumbnails

For image files (`.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`, `.bmp`, `.ico`), if a file named `<image>.thumb` exists beside the image, the listing shows a 32×32 preview in place of the file-type icon. `.thumb` files are hidden from the listing.

## Build

```sh
creo           # or go build -o build/blik .
```

Requires Go 1.26+.
