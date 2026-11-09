package web

import (
	"bytes"
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/json/v2"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"gitverse.ru/zagvozdeen/agenda/internal/store"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

const previewWidth, previewHeight = 1200, 630
const previewCacheLimit = 128

type previewCard struct {
	Kind  string
	ID    string
	Title string
	Board string
}

type previewEntry struct {
	key  string
	data []byte
}

type previewRenderer struct {
	font       *opentype.Font
	background *image.RGBA
	version    string
	mu         sync.Mutex
	cache      map[string]*list.Element
	order      *list.List
}

var previewText = color.RGBA{245, 240, 230, 255}
var previewMuted = color.RGBA{174, 170, 163, 255}
var previewAccent = color.RGBA{255, 144, 110, 255}

func newPreviewRenderer() (*previewRenderer, error) {
	data, err := files.ReadFile("assets/Montserrat-SemiBold.ttf")
	if err != nil {
		return nil, fmt.Errorf("failed to load preview font: %w", err)
	}
	typeface, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse preview font: %w", err)
	}
	background := image.NewRGBA(image.Rect(0, 0, previewWidth, previewHeight))
	draw.Draw(background, background.Bounds(), image.NewUniform(color.RGBA{25, 26, 29, 255}), image.Point{}, draw.Src)
	draw.Draw(background, image.Rect(1124, 128, 1130, 488), image.NewUniform(previewAccent), image.Point{}, draw.Src)
	for i := 0; i < 5; i++ {
		previewLine(background, 1030+float64(i*18), 154, 1030+float64(i*18), 188+float64(i*9), 2, color.NRGBA{255, 144, 110, 70})
	}
	for i := 0; i < 4; i++ {
		angle := float64(i) * math.Pi / 4
		dx, dy := math.Cos(angle)*16, math.Sin(angle)*16
		previewLine(background, 90-dx, 77-dy, 90+dx, 77+dy, 4, previewAccent)
	}
	previewLine(background, 76, 548, 1124, 548, 1, color.NRGBA{255, 255, 255, 35})
	hash := sha256.New()
	hash.Write(data)
	hash.Write(background.Pix)
	fmt.Fprint(hash, "graphite-1")
	return &previewRenderer{font: typeface, background: background, version: fmt.Sprintf("%x", hash.Sum(nil)), cache: make(map[string]*list.Element), order: list.New()}, nil
}

func (p *previewRenderer) key(card previewCard) string {
	data, _ := json.Marshal(card)
	hash := sha256.New()
	fmt.Fprint(hash, p.version)
	hash.Write(data)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func (p *previewRenderer) image(ctx context.Context, card previewCard) ([]byte, error) {
	key := p.key(card)
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if entry := p.cache[key]; entry != nil {
		p.order.MoveToFront(entry)
		return entry.Value.(previewEntry).data, nil
	}
	data, err := p.render(card)
	if err != nil {
		return nil, err
	}
	p.cache[key] = p.order.PushFront(previewEntry{key: key, data: data})
	if p.order.Len() > previewCacheLimit {
		entry := p.order.Back()
		delete(p.cache, entry.Value.(previewEntry).key)
		p.order.Remove(entry)
	}
	return data, nil
}

func (a *Application) setPreview(meta *Metadata, card previewCard) {
	imageURL, _ := url.Parse(meta.URL)
	imageURL.Path = "/og/" + card.ID + ".png"
	imageURL.RawQuery = "v=" + a.previews.key(card)
	meta.Image = imageURL.String()
	meta.ImageAlt = card.ID + " — " + card.Title
}

func (a *Application) servePreview(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/og/"), ".png")
	if !strings.HasSuffix(r.URL.Path, ".png") || (!taskPattern.MatchString(id) && !boardPattern.MatchString(id)) {
		http.NotFound(w, r)
		return
	}
	card, err := a.loadPreview(r.Context(), id)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		if errors.Is(err, store.ErrTaskNotFound) || errors.Is(err, store.ErrBoardNotFound) {
			http.NotFound(w, r)
			return
		}
		a.log.ErrorContext(r.Context(), "Failed to load preview", slog.Any("error", err))
		http.Error(w, "failed to load preview", http.StatusInternalServerError)
		return
	}
	key := a.previews.key(card)
	if version := r.URL.Query().Get("v"); version != "" && version != key {
		http.Redirect(w, r, "/og/"+id+".png?v="+key, http.StatusTemporaryRedirect)
		return
	}
	data, err := a.previews.image(r.Context(), card)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		a.log.ErrorContext(r.Context(), "Failed to render preview", slog.Any("error", err))
		http.Error(w, "failed to render preview", http.StatusInternalServerError)
		return
	}
	if r.URL.Query().Get("v") == key {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("ETag", "\""+key+"\"")
	http.ServeContent(w, r, id+".png", time.Time{}, bytes.NewReader(data))
}

func (a *Application) loadPreview(ctx context.Context, id string) (previewCard, error) {
	if taskPattern.MatchString(id) {
		task, err := a.store.GetTask(ctx, id)
		if err != nil {
			return previewCard{}, err
		}
		board, err := a.store.GetBoard(ctx, task.BoardKey, false)
		if err != nil {
			return previewCard{}, err
		}
		return previewCard{Kind: "task", ID: id, Title: task.Title, Board: board.Name}, nil
	}
	board, err := a.store.GetBoard(ctx, id, false)
	if err != nil {
		return previewCard{}, err
	}
	return previewCard{Kind: "board", ID: id, Title: board.Name}, nil
}

func (p *previewRenderer) render(card previewCard) ([]byte, error) {
	dst := image.NewRGBA(p.background.Bounds())
	copy(dst.Pix, p.background.Pix)
	face, err := p.face(28)
	if err != nil {
		return nil, err
	}
	defer face.Close()
	previewString(dst, face, "agenda", 123, 87, previewText)
	label, footer := "ЗАДАЧА  /  "+card.ID, card.Board
	if card.Kind == "board" {
		label, footer = "ДОСКА  /  "+card.ID, "Задачи, идеи и следующие шаги"
	}
	previewString(dst, face, label, 76, 178, previewAccent)
	previewString(dst, face, "ЗАДАЧИ И ИДЕИ", 1124-font.MeasureString(face, "ЗАДАЧИ И ИДЕИ").Ceil(), 87, previewMuted)
	lines, size, titleFace, err := p.fit(card.Title)
	if err != nil {
		return nil, err
	}
	defer titleFace.Close()
	lineHeight := float64(size) * 1.17
	blockHeight := float64(len(lines)-1)*lineHeight + float64(size)
	baseline := 218 + (280-blockHeight)/2 + float64(titleFace.Metrics().Ascent)/64
	for _, line := range lines {
		previewString(dst, titleFace, line, 72, int(math.Round(baseline)), previewText)
		baseline += lineHeight
	}
	previewString(dst, face, previewEllipsis(face, footer, 810), 76, 595, previewMuted)
	previewString(dst, face, card.ID, 1124-font.MeasureString(face, card.ID).Ceil(), 595, previewMuted)
	var data bytes.Buffer
	if err = png.Encode(&data, dst); err != nil {
		return nil, fmt.Errorf("failed to encode preview: %w", err)
	}
	return data.Bytes(), nil
}

func (p *previewRenderer) face(size int) (*opentype.Face, error) {
	face, err := opentype.NewFace(p.font, &opentype.FaceOptions{Size: float64(size), DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, fmt.Errorf("failed to create preview font face: %w", err)
	}
	return face.(*opentype.Face), nil
}

func (p *previewRenderer) fit(value string) ([]string, int, *opentype.Face, error) {
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		value = "Без названия"
	}
	maximum := 88
	if utf8.RuneCountInString(value) < 14 {
		maximum = 116
	}
	for size := maximum; size >= 52; size -= 2 {
		face, err := p.face(size)
		if err != nil {
			return nil, 0, nil, err
		}
		lines := previewWrap(face, value, 1048)
		blockHeight := float64(len(lines)-1)*float64(size)*1.17 + float64(size)
		if len(lines) <= 4 && blockHeight <= 280 {
			return lines, size, face, nil
		}
		if size == 52 {
			lines = lines[:4]
			lines[3] = previewEllipsis(face, lines[3]+"…", 1048)
			return lines, size, face, nil
		}
		face.Close()
	}
	return nil, 0, nil, fmt.Errorf("failed to fit preview title")
}

func previewWrap(face font.Face, value string, maxWidth int) []string {
	var lines []string
	current := ""
	for _, word := range strings.Fields(value) {
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if font.MeasureString(face, candidate).Ceil() <= maxWidth {
			current = candidate
			continue
		}
		if current != "" {
			lines = append(lines, current)
			current = ""
		}
		for _, symbol := range word {
			candidate = current + string(symbol)
			if font.MeasureString(face, candidate).Ceil() > maxWidth && current != "" {
				lines = append(lines, current)
				current = ""
			}
			current += string(symbol)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func previewEllipsis(face font.Face, value string, maxWidth int) string {
	if font.MeasureString(face, value).Ceil() <= maxWidth {
		return value
	}
	runes := []rune(strings.TrimRightFunc(value, unicode.IsSpace))
	for len(runes) > 0 && font.MeasureString(face, string(runes)+"…").Ceil() > maxWidth {
		runes = runes[:len(runes)-1]
	}
	return strings.TrimRightFunc(string(runes), unicode.IsSpace) + "…"
}

func previewString(dst draw.Image, face font.Face, value string, x, y int, c color.Color) {
	drawer := font.Drawer{Dst: dst, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x, y)}
	drawer.DrawString(value)
}

func previewLine(dst draw.Image, x1, y1, x2, y2, thickness float64, c color.Color) {
	dx, dy := x2-x1, y2-y1
	length := math.Hypot(dx, dy)
	nx, ny := -dy/length*thickness/2, dx/length*thickness/2
	bounds := dst.Bounds()
	raster := vector.NewRasterizer(bounds.Dx(), bounds.Dy())
	raster.MoveTo(float32(x1+nx), float32(y1+ny))
	raster.LineTo(float32(x2+nx), float32(y2+ny))
	raster.LineTo(float32(x2-nx), float32(y2-ny))
	raster.LineTo(float32(x1-nx), float32(y1-ny))
	raster.ClosePath()
	raster.Draw(dst, bounds, image.NewUniform(c), image.Point{})
}
