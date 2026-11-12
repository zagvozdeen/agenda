package web

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"gitverse.ru/zagvozdeen/agenda/internal/config"
	"gitverse.ru/zagvozdeen/agenda/internal/store"
)

//go:embed templates static assets
var files embed.FS

var boardPattern = regexp.MustCompile("^[A-Z]{3}$")
var taskPattern = regexp.MustCompile("^[A-Z]{3}-[1-9][0-9]{0,9}$")

type Application struct {
	store     store.Storage
	log       *slog.Logger
	templates *template.Template
	assets    http.Handler
	version   string
	publicURL string
	previews  *previewRenderer
}

type Metadata struct {
	Title       string
	Description string
	URL         string
	Image       string
	ImageAlt    string
}

type Page struct {
	Kind      string
	Version   string
	Meta      Metadata
	Board     *BoardPage
	Task      *TaskPage
	Error     string
	ErrorCode int
}

func New(cfg *config.Config, log *slog.Logger, storage store.Storage) (*Application, error) {
	publicURL := strings.TrimRight(cfg.App.PublicURL, "/")
	if publicURL != "" {
		u, err := url.Parse(publicURL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return nil, fmt.Errorf("invalid public URL")
		}
	}
	templates, err := template.New("pages").Funcs(template.FuncMap{
		"status":        statusLabel,
		"statusClass":   statusClass,
		"taskType":      typeLabel,
		"priority":      priorityLabel,
		"date":          func(value time.Time) string { return value.UTC().Format("02.01.2006, 15:04 UTC") },
		"isoDate":       func(value time.Time) string { return value.Format(time.RFC3339) },
		"taskCount":     taskCount,
		"taskWord":      taskWord,
		"markdown":      renderMarkdown,
		"codeLinkLabel": codeLinkLabel,
		"shortSHA": func(value *string) string {
			if value == nil {
				return ""
			}
			return (*value)[:min(12, len(*value))]
		},
	}).ParseFS(files, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse web templates: %w", err)
	}
	assets, err := fs.Sub(files, "static")
	if err != nil {
		return nil, fmt.Errorf("failed to load web assets: %w", err)
	}
	previews, err := newPreviewRenderer()
	if err != nil {
		return nil, err
	}
	hash := sha256.New()
	if err = fs.WalkDir(assets, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		content, err := fs.ReadFile(assets, path)
		if err != nil {
			return err
		}
		hash.Write([]byte(path))
		hash.Write(content)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to fingerprint web assets: %w", err)
	}
	return &Application{
		store: storage, log: log, templates: templates,
		assets: http.FileServerFS(assets), version: fmt.Sprintf("%x", hash.Sum(nil))[:12], publicURL: publicURL, previews: previews,
	}, nil
}

func (a *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/static/") {
		a.serveAsset(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	if strings.HasPrefix(r.URL.Path, "/og/") {
		a.servePreview(w, r)
		return
	}
	if r.URL.Path == "/robots.txt" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if r.Method != http.MethodHead {
			fmt.Fprint(w, "User-agent: *\nAllow: /\nDisallow: /mcp\n")
		}
		return
	}
	var page *Page
	var err error
	if r.URL.Path == "/" {
		page, err = a.boardPage(r)
	} else if id := strings.TrimPrefix(r.URL.Path, "/"); taskPattern.MatchString(id) {
		page, err = a.taskPage(r, id)
	} else if boardPattern.MatchString(strings.TrimPrefix(r.URL.Path, "/")) {
		page, err = a.boardPage(r)
	} else {
		a.errorPage(w, r, http.StatusNotFound, "Page not found")
		return
	}
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		switch {
		case errors.Is(err, store.ErrBoardNotFound), errors.Is(err, store.ErrTaskNotFound):
			a.errorPage(w, r, http.StatusNotFound, "Task or board not found")
		case errors.Is(err, errInvalidFilter):
			a.errorPage(w, r, http.StatusBadRequest, "Invalid filters")
		default:
			a.log.ErrorContext(r.Context(), "Failed to render web page", slog.Any("error", err))
			a.errorPage(w, r, http.StatusInternalServerError, "Could not load this page")
		}
		return
	}
	name := "page"
	if page.Kind == "board" {
		w.Header().Add("Vary", "X-Agenda-Partial")
		if r.Header.Get("X-Agenda-Partial") == "board" {
			name = "workspace"
		}
	}
	a.render(w, r, http.StatusOK, name, page)
}

func (a *Application) serveAsset(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/static/")
	switch name {
	case "app.css", "app.js", "board.js", "filters.js", "select.js", "dates.js", "icons.svg", "favicon.svg", "social-preview.png":
	default:
		http.NotFound(w, r)
		return
	}
	if r.URL.Query().Get("v") == a.version {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	http.StripPrefix("/static/", a.assets).ServeHTTP(w, r)
}

func (a *Application) render(w http.ResponseWriter, r *http.Request, code int, name string, page *Page) {
	page.Version = a.version
	var body bytes.Buffer
	if err := a.templates.ExecuteTemplate(&body, name, page); err != nil {
		a.log.ErrorContext(r.Context(), "Failed to execute web template", slog.Any("error", err))
		http.Error(w, "failed to render page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	if r.Method != http.MethodHead {
		if _, err := body.WriteTo(w); err != nil {
			a.log.DebugContext(r.Context(), "Failed to write web response", slog.Any("error", err))
		}
	}
}

func (a *Application) errorPage(w http.ResponseWriter, r *http.Request, code int, message string) {
	page := &Page{Kind: "error", Error: message, ErrorCode: code}
	page.Meta = a.metadata(r, message+" — Agenda", message, r.URL.Path, "")
	w.Header().Set("X-Robots-Tag", "noindex")
	a.render(w, r, code, "page", page)
}

func (a *Application) metadata(r *http.Request, title, description, path, query string) Metadata {
	origin := a.publicURL
	if origin == "" {
		scheme, host := "http", r.Host
		if r.TLS != nil {
			scheme = "https"
		}
		if remote, _, err := net.SplitHostPort(r.RemoteAddr); err == nil && net.ParseIP(remote).IsLoopback() {
			if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" && !strings.ContainsAny(forwarded, ",/\\ \t\r\n@?#") {
				host = forwarded
			}
			if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded == "https" || forwarded == "http" {
				scheme = forwarded
			}
		}
		origin = scheme + "://" + host
	}
	canonical, _ := url.Parse(origin)
	if canonical == nil || canonical.Host == "" {
		canonical = &url.URL{Scheme: "http", Host: r.Host}
	}
	canonical.Path, canonical.RawQuery = path, query
	return Metadata{Title: title, Description: excerpt(description, 180), URL: canonical.String(), Image: origin + "/static/social-preview.png?v=" + a.version, ImageAlt: "Agenda — Меньше шума. Больше сделанного."}
}
