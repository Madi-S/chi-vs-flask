package main

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/exp/slog"

	"main/cmd/internal/config"
	"main/cmd/internal/http-server/handlers/redirect"
	"main/cmd/internal/http-server/handlers/url/save"
	mwLogger "main/cmd/internal/http-server/middleware/logger"
	"main/cmd/internal/lib/logger/handlers/slogpretty"
	"main/cmd/internal/lib/logger/sl"
	"main/cmd/internal/storage/sqlite"
)

const (
	envDev   = "dev"
	envProd  = "prod"
	envLocal = "local"
)

func main() {
	// init config: cleanenv
	cfg := config.MustLoad()

	// init logger: slog
	log := setupLogger(cfg.Env)
	log = log.With(slog.String("env", cfg.Env))

	// init storage: sqlite
	storage, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		log.Error("Failed to initialize storage", sl.Err(err))
		os.Exit(69)
	}

	// init router: chi, chi-render
	router := chi.NewRouter()

	// init middlewares
	router.Use(mwLogger.New(log))
	router.Use(middleware.RealIP)
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/api/v1/url", func(r chi.Router) {
		r.Use(middleware.BasicAuth("url-shortener", map[string]string{
			cfg.HTTPServer.User: cfg.HTTPServer.Password,
			"JT":                "veryLongPassword",
			"floppy":            "veryLongAssPassword",
		}))

		r.Post("/", save.New(log, storage))
		// The rest is REST
		// r.Patch("/{alias}", update.New(log, storage))
		// r.Delete("/{alias}", delete.New(log, storage))
	})

	router.Get("/{alias}", redirect.New(log, storage))

	// init run server
	log.Info("Starting server", slog.String("address", cfg.Address))

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Error("Failed to start server")
	}

	log.Error("Server is not running, something went wrong")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
		// log = slog.New(
		// 	slog.NewTextHandler(
		// 		os.Stdout,
		// 		&slog.HandlerOptions{Level: slog.LevelDebug},
		// 	),
		// )
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(
				os.Stdout,
				&slog.HandlerOptions{Level: slog.LevelDebug},
			),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(
				os.Stdout,
				&slog.HandlerOptions{Level: slog.LevelInfo},
			),
		)
	default:
		log = slog.New(
			slog.NewTextHandler(
				os.Stdout,
				&slog.HandlerOptions{Level: slog.LevelDebug},
			),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
