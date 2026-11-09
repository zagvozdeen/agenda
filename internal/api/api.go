package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gitverse.ru/zagvozdeen/agenda/internal/config"
	"gitverse.ru/zagvozdeen/agenda/internal/store"
	"golang.org/x/time/rate"
)

type API struct {
	cfg   *config.Config
	log   *slog.Logger
	store store.Storage
	mcp   *mcp.Server
	pages http.Handler
}

func NewApplication(cfg *config.Config, log *slog.Logger, storage store.Storage, pages http.Handler) *API {
	a := &API{
		cfg:   cfg,
		log:   log,
		store: storage,
		pages: pages,
		mcp: mcp.NewServer(&mcp.Implementation{
			Name:    "agenda",
			Version: "0.1.0",
		}, &mcp.ServerOptions{
			Logger:       log,
			Instructions: "Agenda task tracker. Reading is public; whoami and writes require a personal Bearer token. Board owners manage boards and members; board members manage tasks, comments and sprints. Search filters use current task fields. Nullable update fields accept null to clear their values.",
		}),
	}
	a.registerUsers()
	a.registerBoards()
	a.registerSprints()
	a.registerTasks()
	a.registerComments()
	a.registerActivity()
	return a
}

func (a *API) Handler() http.Handler {
	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server { return a.mcp }, &mcp.StreamableHTTPOptions{
		Stateless:           true,
		JSONResponse:        true,
		Logger:              a.log,
		MaxRequestBodyBytes: 1 << 20,
	})
	limiter := rate.NewLimiter(20, 40)

	mux := http.NewServeMux()
	mux.Handle("/mcp", a.rateLimit(limiter, http.NewCrossOriginProtection().Handler(a.authenticate(handler))))
	mux.Handle("/", a.pages)
	return mux
}

func (a *API) Run(ctx context.Context) error {
	addr := net.JoinHostPort(a.cfg.App.Host, a.cfg.App.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	server := &http.Server{
		Handler:           a.Handler(),
		ErrorLog:          slog.NewLogLogger(a.log.Handler(), slog.LevelError),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		a.log.InfoContext(ctx, "Server started", slog.String("addr", addr))
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if err = server.Shutdown(shutdownCtx); err != nil {
			if closeErr := server.Close(); closeErr != nil {
				err = errors.Join(closeErr, err)
			}
			return fmt.Errorf("failed to shutdown server: %w", err)
		}
	case err = <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("failed to serve HTTP: %w", err)
		}
	}
	a.log.InfoContext(ctx, "Application has successfully stopped")
	return nil
}
