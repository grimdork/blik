package content

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/grimdork/climate/fx"
)

var (
	iconCache   map[string]string
	iconOnce    sync.Once
	iconMu      sync.RWMutex
	iconDir     string
	defaultIcon string
)

var extIcon = map[string]string{
	".7z":      "file-zip",
	".adoc":    "file-text",
	".avi":     "video",
	".bmp":     "file-type-bmp",
	".bz2":     "file-zip",
	".cfg":     "file-settings",
	".conf":    "file-settings",
	".css":     "file-type-css",
	".csv":     "file-type-csv",
	".db":      "file-database",
	".dmg":     "file-unknown",
	".doc":     "file-type-doc",
	".docx":    "file-type-doc",
	".env":     "file-settings",
	".exe":     "file-unknown",
	".flac":    "music",
	".gif":     "photo",
	".go":      "file-code",
	".gz":      "file-zip",
	".htm":     "file-type-html",
	".html":    "file-type-html",
	".ico":     "photo",
	".ini":     "file-settings",
	".iso":     "file-unknown",
	".java":    "file-code",
	".jpeg":    "file-type-jpg",
	".jpg":     "file-type-jpg",
	".js":      "file-type-js",
	".json":    "file-code",
	".jsx":     "file-type-jsx",
	".m4a":     "music",
	".md":      "file-text",
	".mkv":     "video",
	".mov":     "video",
	".mp3":     "music",
	".mp4":     "video",
	".ogg":     "music",
	".pdf":     "file-type-pdf",
	".php":     "file-type-php",
	".pl":      "file-code",
	".png":     "file-type-png",
	".ppt":     "file-type-ppt",
	".pptx":    "file-type-ppt",
	".py":      "file-code",
	".rar":     "file-zip",
	".rb":      "file-code",
	".rs":      "file-type-rs",
	".rst":     "file-text",
	".sass":    "file-type-css",
	".scss":    "file-type-css",
	".sh":      "file-code",
	".sql":     "file-type-sql",
	".sqlite":  "file-database",
	".sqlite3": "file-database",
	".svg":     "file-type-svg",
	".tar":     "file-zip",
	".tgz":     "file-zip",
	".toml":    "file-code",
	".ts":      "file-type-ts",
	".tsx":     "file-type-tsx",
	".txt":     "file-text",
	".vue":     "file-type-vue",
	".wav":     "music",
	".webm":    "video",
	".webp":    "photo",
	".xls":     "file-type-xls",
	".xlsx":    "file-type-xls",
	".xml":     "file-type-xml",
	".xz":      "file-zip",
	".yaml":    "file-code",
	".yml":     "file-code",
	".zip":     "file-zip",
}

var imageExts = map[string]bool{
	".bmp":  true,
	".gif":  true,
	".ico":  true,
	".jpeg": true,
	".jpg":  true,
	".png":  true,
	".webp": true,
}

var mediaExts = map[string]bool{
	".avi":  true,
	".flac": true,
	".m4a":  true,
	".m4v":  true,
	".mkv":  true,
	".mov":  true,
	".mp3":  true,
	".mp4":  true,
	".ogg":  true,
	".wav":  true,
	".webm": true,
}

var audioExts = map[string]bool{
	".flac": true,
	".m4a":  true,
	".mp3":  true,
	".ogg":  true,
	".wav":  true,
}

var archiveExts = map[string]bool{
	".tar":    true,
	".tar.gz": true,
	".tgz":    true,
	".zip":    true,
}

func initIconCache(dir string) {
	iconDir = dir
	iconOnce.Do(func() {
		iconCache = make(map[string]string)
		defaultIcon = loadSVG("file-unknown")
		if defaultIcon == "" {
			defaultIcon = `<!-- missing -->`
		}
	})
}

func iconName(name string, isDir bool) string {
	if isDir {
		return "folder"
	}
	ext := strings.ToLower(filepath.Ext(name))
	if icon, ok := extIcon[ext]; ok {
		return icon
	}
	return "file-unknown"
}

func iconSVG(name string, isDir bool) string {
	key := iconName(name, isDir)

	iconMu.RLock()
	if cached, ok := iconCache[key]; ok {
		iconMu.RUnlock()
		return cached
	}
	iconMu.RUnlock()

	svg := loadSVG(key)

	iconMu.Lock()
	iconCache[key] = svg
	iconMu.Unlock()
	return svg
}

func loadSVG(name string) string {
	path := filepath.Join(iconDir, name+".svg")
	b, err := os.ReadFile(path)
	if err != nil {
		fx.Fprintln(os.Stderr, "{logstamp} {danger}icons:{@} {} — {}", path, err)
		return defaultIcon
	}
	return string(b)
}

func isImage(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return imageExts[ext]
}

func isMedia(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return mediaExts[ext]
}

func isArchive(name string) bool {
	n := strings.ToLower(name)
	if strings.HasSuffix(n, ".tar.gz") || strings.HasSuffix(n, ".tgz") {
		return true
	}
	ext := filepath.Ext(n)
	return archiveExts[ext]
}

func thumbPath(fullPath string) string {
	return fullPath + ".thumb"
}

func thumbExists(fullPath string) bool {
	_, err := os.Stat(thumbPath(fullPath))
	return err == nil
}
