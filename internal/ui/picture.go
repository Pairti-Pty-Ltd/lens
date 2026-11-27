package ui

import (
	"bytes"
	"encoding/base64"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

const (
	maxImageBytes = 8 << 20
	maxImageEdge  = 4096
	maxRemote     = 8
)

var imageClient = &http.Client{Timeout: time.Second}

type fitKey struct{ w, h int }

type cachedPic struct {
	src  image.Image
	fit  map[fitKey]image.Image
	fail bool
}

// loadPicture decodes raw, which is a path relative to the document
// directory, a file URL, a data URL, or an http(s) URL.
// A failure is cached until the next SetSource. The returned image is
// the source bitmap, already bounded to maxImageEdge.
func (p *Page) loadPicture(raw string) image.Image {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if p.pics == nil {
		p.pics = map[string]*cachedPic{}
	}
	if c, ok := p.pics[raw]; ok {
		if c.fail {
			return nil
		}
		return c.src
	}
	if len(p.pics) > 64 {
		p.pics = map[string]*cachedPic{}
	}
	img, err := p.decodePicture(raw)
	c := &cachedPic{}
	if err != nil || img == nil {
		c.fail = true
	} else {
		c.src = boundPicture(img)
	}
	p.pics[raw] = c
	return c.src
}

func (p *Page) fitted(raw string, src image.Image, w, h int) image.Image {
	if src == nil {
		return nil
	}
	b := src.Bounds()
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if b.Dx() == w && b.Dy() == h {
		return src
	}
	c := p.pics[raw]
	if c == nil {
		return scalePicture(src, w, h)
	}
	if c.fit == nil {
		c.fit = map[fitKey]image.Image{}
	}
	k := fitKey{w, h}
	if img, ok := c.fit[k]; ok {
		return img
	}
	img := scalePicture(src, w, h)
	c.fit[k] = img
	return img
}

func (p *Page) decodePicture(raw string) (image.Image, error) {
	mode, loc := resolvePicture(p.dir, raw)
	switch mode {
	case "data":
		return decodeData(loc)
	case "http":
		if p.remoteLeft <= 0 {
			return nil, os.ErrInvalid
		}
		p.remoteLeft--
		return fetchPicture(loc)
	default:
		if loc == "" {
			return nil, os.ErrNotExist
		}
		return decodeFile(loc)
	}
}

func resolvePicture(dir, raw string) (mode, loc string) {
	raw = strings.TrimSpace(raw)
	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "data:image/"):
		return "data", raw
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"):
		return "http", raw
	case strings.HasPrefix(lower, "file:"):
		u, err := url.Parse(raw)
		if err != nil {
			return "file", ""
		}
		return "file", u.Path
	}
	u := raw
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		u = u[:i]
	}
	if dec, err := url.PathUnescape(u); err == nil {
		u = dec
	}
	u = filepath.FromSlash(u)
	if filepath.IsAbs(u) {
		return "file", u
	}
	if dir == "" {
		return "file", ""
	}
	return "file", filepath.Clean(filepath.Join(dir, u))
}

func decodeFile(path string) (image.Image, error) {
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() || st.Size() == 0 || st.Size() > maxImageBytes {
		if err == nil {
			err = os.ErrInvalid
		}
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(io.LimitReader(f, maxImageBytes))
	return img, err
}

func fetchPicture(raw string) (image.Image, error) {
	resp, err := imageClient.Get(raw)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, os.ErrInvalid
	}
	img, _, err := image.Decode(io.LimitReader(resp.Body, maxImageBytes))
	return img, err
}

func decodeData(raw string) (image.Image, error) {
	comma := strings.Index(raw, ",")
	if comma < 0 {
		return nil, os.ErrInvalid
	}
	meta, payload := raw[:comma], raw[comma+1:]
	var r io.Reader
	if strings.Contains(strings.ToLower(meta), ";base64") {
		b, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			b, err = base64.RawStdEncoding.DecodeString(payload)
			if err != nil {
				return nil, err
			}
		}
		r = bytes.NewReader(b)
	} else {
		s, err := url.PathUnescape(payload)
		if err != nil {
			return nil, err
		}
		r = strings.NewReader(s)
	}
	img, _, err := image.Decode(r)
	return img, err
}

func boundPicture(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	edge := w
	if h > edge {
		edge = h
	}
	if edge <= maxImageEdge || edge < 1 {
		return img
	}
	s := float64(maxImageEdge) / float64(edge)
	return scalePicture(img, int(float64(w)*s), int(float64(h)*s))
}

func scalePicture(src image.Image, w, h int) image.Image {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func fitPicture(nw, nh int, maxW, maxH float32) (int, int) {
	w := float32(nw)
	h := float32(nh)
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if w > maxW {
		h = h * maxW / w
		w = maxW
	}
	if h > maxH {
		w = w * maxH / h
		h = maxH
	}
	dw := int(w + 0.5)
	dh := int(h + 0.5)
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}
	return dw, dh
}

func imageLabel(alt, raw string) string {
	alt = strings.TrimSpace(alt)
	if alt != "" {
		return alt
	}
	return imageName(raw)
}

func imageName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(strings.ToLower(raw), "data:") {
		return "Image"
	}
	u := raw
	if strings.Contains(u, "://") {
		if parsed, err := url.Parse(u); err == nil && parsed.Path != "" {
			u = parsed.Path
		}
	}
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		u = u[:i]
	}
	if dec, err := url.PathUnescape(u); err == nil {
		u = dec
	}
	base := filepath.Base(filepath.FromSlash(u))
	if base == "" || base == "." || base == "/" || base == string(filepath.Separator) {
		return "Image"
	}
	return base
}
