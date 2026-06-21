package content

import (
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"blik/blikconfig"
	"github.com/fsnotify/fsnotify"
	"github.com/grimdork/climate/fx"
)

type thumbSize struct {
	width  int
	height int
	crop   bool
}

func parseThumbSize(s string) (thumbSize, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return thumbSize{}, fmt.Errorf("empty size")
	}

	if strings.HasSuffix(s, "w") {
		n, err := strconv.Atoi(s[:len(s)-1])
		if err != nil || n <= 0 {
			return thumbSize{}, fmt.Errorf("invalid width: %q", s)
		}
		return thumbSize{width: n}, nil
	}

	if strings.HasSuffix(s, "h") {
		n, err := strconv.Atoi(s[:len(s)-1])
		if err != nil || n <= 0 {
			return thumbSize{}, fmt.Errorf("invalid height: %q", s)
		}
		return thumbSize{height: n}, nil
	}

	parts := strings.SplitN(s, "x", 2)
	if len(parts) == 2 {
		w, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		h, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
			return thumbSize{}, fmt.Errorf("invalid dimensions: %q", s)
		}
		return thumbSize{width: w, height: h, crop: true}, nil
	}

	return thumbSize{}, fmt.Errorf("invalid thumbsize: %q (use e.g. 128w, 128h, 128x128)", s)
}

func bilinearScale(src image.Image, dst *image.RGBA) {
	b := dst.Bounds()
	sb := src.Bounds()
	srcW := sb.Dx()
	srcH := sb.Dy()
	dstW := b.Dx()
	dstH := b.Dy()

	for y := 0; y < dstH; y++ {
		for x := 0; x < dstW; x++ {
			sx := float64(x) * float64(srcW) / float64(dstW)
			sy := float64(y) * float64(srcH) / float64(dstH)

			ix := int(math.Floor(sx))
			iy := int(math.Floor(sy))
			dx := sx - float64(ix)
			dy := sy - float64(iy)

			ix1 := min(ix, srcW-1)
			ix2 := min(ix+1, srcW-1)
			iy1 := min(iy, srcH-1)
			iy2 := min(iy+1, srcH-1)

			c00 := src.At(ix1+sb.Min.X, iy1+sb.Min.Y)
			c10 := src.At(ix2+sb.Min.X, iy1+sb.Min.Y)
			c01 := src.At(ix1+sb.Min.X, iy2+sb.Min.Y)
			c11 := src.At(ix2+sb.Min.X, iy2+sb.Min.Y)

			r00, g00, b00, a00 := c00.RGBA()
			r10, g10, b10, a10 := c10.RGBA()
			r01, g01, b01, a01 := c01.RGBA()
			r11, g11, b11, a11 := c11.RGBA()

			r := lerp(lerp(r00, r10, dx), lerp(r01, r11, dx), dy)
			g := lerp(lerp(g00, g10, dx), lerp(g01, g11, dx), dy)
			blue := lerp(lerp(b00, b10, dx), lerp(b01, b11, dx), dy)
			a := lerp(lerp(a00, a10, dx), lerp(a01, a11, dx), dy)

			off := dst.PixOffset(x+b.Min.X, y+b.Min.Y)
			dst.Pix[off+0] = uint8(r >> 8)
			dst.Pix[off+1] = uint8(g >> 8)
			dst.Pix[off+2] = uint8(blue >> 8)
			dst.Pix[off+3] = uint8(a >> 8)
		}
	}
}

func lerp(a, b uint32, t float64) uint32 {
	return uint32(float64(a)*(1-t) + float64(b)*t)
}

func scaleAndCrop(src image.Image, sz thumbSize) image.Image {
	sb := src.Bounds()
	srcW := sb.Dx()
	srcH := sb.Dy()

	switch {
	case sz.width > 0 && sz.height == 0 && !sz.crop:
		if srcW <= sz.width {
			return src
		}
		dstH := srcH * sz.width / srcW
		if dstH < 1 {
			dstH = 1
		}
		dst := image.NewRGBA(image.Rect(0, 0, sz.width, dstH))
		bilinearScale(src, dst)
		return dst

	case sz.height > 0 && sz.width == 0 && !sz.crop:
		if srcH <= sz.height {
			return src
		}
		dstW := srcW * sz.height / srcH
		if dstW < 1 {
			dstW = 1
		}
		dst := image.NewRGBA(image.Rect(0, 0, dstW, sz.height))
		bilinearScale(src, dst)
		return dst

	case sz.width > 0 && sz.height > 0 && sz.crop:
		if srcW <= sz.width && srcH <= sz.height {
			return centreCrop(src, sz.width, sz.height)
		}

		var scaled *image.RGBA
		if srcW <= srcH {
			dstH := srcH * sz.width / srcW
			if dstH < sz.height {
				dstH = sz.height
			}
			scaled = image.NewRGBA(image.Rect(0, 0, sz.width, dstH))
			bilinearScale(src, scaled)
		} else {
			dstW := srcW * sz.height / srcH
			if dstW < sz.width {
				dstW = sz.width
			}
			scaled = image.NewRGBA(image.Rect(0, 0, dstW, sz.height))
			bilinearScale(src, scaled)
		}
		return centreCrop(scaled, sz.width, sz.height)
	}

	return src
}

func centreCrop(img image.Image, w, h int) image.Image {
	b := img.Bounds()
	sw := b.Dx()
	sh := b.Dy()
	cx := (sw - w) / 2
	cy := (sh - h) / 2
	if cx < 0 {
		cx = 0
	}
	if cy < 0 {
		cy = 0
	}
	cw := min(w, sw)
	ch := min(h, sh)

	dst := image.NewRGBA(image.Rect(0, 0, cw, ch))
	for y := 0; y < ch; y++ {
		for x := 0; x < cw; x++ {
			dst.Set(x, y, img.At(cx+x, cy+y))
		}
	}
	return dst
}

func generateThumb(srcPath, dstPath string, sz thumbSize) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	img, _, err := image.Decode(src)
	if err != nil {
		return err
	}

	thumb := scaleAndCrop(img, sz)

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	return jpeg.Encode(dst, thumb, &jpeg.Options{Quality: 50})
}

func isThumbnailCandidate(name string) bool {
	return isImage(name) && !strings.HasSuffix(strings.ToLower(name), ".ico")
}

func startThumbWorkers(root string, cfg *blikconfig.Config) {
	sz, err := parseThumbSize(cfg.ThumbSize)
	if err != nil {
		fx.Fprintln(os.Stderr, "{logstamp} {danger}thumbs:{@} invalid thumbsize: {} — {}", cfg.ThumbSize, err)
		return
	}

	n := cfg.ThumbWorkers
	if n < 1 {
		n = 1
	}

	jobs := make(chan string, 100)

	for i := 0; i < n; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fx.Fprintln(os.Stderr, "{logstamp} {danger}thumb worker panic:{@} {}", r)
				}
			}()
			for src := range jobs {
				dst := src + ".thumb"
				if err := generateThumb(src, dst, sz); err != nil {
					fx.Fprintln(os.Stderr, "{logstamp} {danger}thumbs:{@} {} — {}", src, err)
				} else {
					fx.Println("{logstamp} {success}thumbs:{@} {}", src)
				}
			}
		}()
	}

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || strings.HasSuffix(info.Name(), ".thumb") || !isThumbnailCandidate(info.Name()) {
			return nil
		}

		thumbFn := path + ".thumb"
		if tfi, stErr := os.Stat(thumbFn); stErr == nil && !tfi.ModTime().Before(info.ModTime()) {
			return nil
		}

		jobs <- path
		return nil
	})

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fx.Fprintln(os.Stderr, "{logstamp} {warning}thumbs:{@} watcher unavailable: {}", err)
		return
	}
	defer watcher.Close()

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			watcher.Add(path)
		}
		return nil
	})

	var timerMu sync.Mutex
	timers := make(map[string]*time.Timer)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if strings.HasSuffix(event.Name, ".thumb") {
				continue
			}
			if fi, stErr := os.Stat(event.Name); stErr == nil && fi.IsDir() {
				_ = watcher.Add(event.Name)
				continue
			}
			if !isThumbnailCandidate(event.Name) {
				continue
			}
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
				timerMu.Lock()
				if t, ok := timers[event.Name]; ok {
					t.Reset(500 * time.Millisecond)
				} else {
					timers[event.Name] = time.AfterFunc(500*time.Millisecond, func() {
						timerMu.Lock()
						delete(timers, event.Name)
						timerMu.Unlock()
						jobs <- event.Name
					})
				}
				timerMu.Unlock()
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fx.Fprintln(os.Stderr, "{logstamp} {warning}watcher error:{@} {}", err)
		}
	}
}
