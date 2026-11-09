package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"gitverse.ru/zagvozdeen/agenda/internal/config"
	"gitverse.ru/zagvozdeen/agenda/internal/db"
	"gitverse.ru/zagvozdeen/agenda/internal/logger"
	"gitverse.ru/zagvozdeen/agenda/internal/store"
)

type application struct {
	cfg        *config.Config
	log        *slog.Logger
	pool       *pgxpool.Pool
	store      store.Storage
	stop       func()
	runServer  func(context.Context, *config.Config, *slog.Logger, store.Storage) error
}

func Run(ctx context.Context, serve func(context.Context, *config.Config, *slog.Logger, store.Storage) error) error {
	a := &application{runServer: serve}
	defer a.close()

	root := &cobra.Command{
		Use:               "agenda",
		Short:             "Minimal task tracker for AI agents",
		Version:           "0.1.0",
		SilenceErrors:     true,
		SilenceUsage:      true,
		Args:              cobra.NoArgs,
		PersistentPreRunE: a.init,
		RunE:              a.serve,
	}
	root.AddCommand(a.server(), a.users(), a.tokens())

	return root.ExecuteContext(ctx)
}

func (a *application) init(cmd *cobra.Command, _ []string) error {
	for current := cmd; current != nil; current = current.Parent() {
		switch current.Name() {
		case "help", "completion", cobra.ShellCompRequestCmd:
			return nil
		}
	}
	err := cmd.ValidateRequiredFlags()
	if err != nil {
		return err
	}
	a.cfg, err = config.New()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	a.log, a.stop, err = logger.New(a.cfg)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	a.pool, err = db.New(cmd.Context(), a.cfg, a.log)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	a.store = store.New(a.pool, a.log)
	return nil
}

func (a *application) close() {
	if a.pool != nil {
		a.pool.Close()
	}
	if a.stop != nil {
		a.stop()
	}
}

func (a *application) serve(cmd *cobra.Command, _ []string) error {
	return a.runServer(cmd.Context(), a.cfg, a.log, a.store)
}

func (a *application) server() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP MCP server",
		Args:  cobra.NoArgs,
		RunE:  a.serve,
	}
}
