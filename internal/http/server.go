package http

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	server    *http.Server
	mu        *sync.RWMutex
	isRunning bool
	port      int
	logger    *slog.Logger

	userHandler  *UserHandler
	teamHandler  *TeamHandler
	prHandler    *PrHandler
	statsHandler *StatsHandler
}

func NewServer(logger *slog.Logger, db *gorm.DB) *Server {
	return &Server{
		userHandler:  NewUserHandler(logger, db),
		teamHandler:  NewTeamHandler(logger, db),
		prHandler:    NewPrHandler(logger, db),
		statsHandler: NewStatsHandler(logger, db),
		logger:       logger,

		port: 8080,
		mu:   &sync.RWMutex{},
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("server is already running")
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(s.loggingMiddleware)

	s.registerRoutes(router)

	s.server = &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", s.port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("Starting HTTP server on", "port", s.port)

	errCh := make(chan error, 1)
	go func() {
		s.mu.Lock()
		s.isRunning = true
		s.mu.Unlock()

		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	s.mu.Unlock()

	select {
	case err := <-errCh:
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
		return fmt.Errorf("server failed to start: %w", err)
	case <-ctx.Done():
		return s.Stop(ctx, 10*time.Second)
	}
}

func (s *Server) registerRoutes(router *chi.Mux) {
	router.Route("/users", func(r chi.Router) {
		r.Post("/setIsActive", s.userHandler.SetUserIsActive)
		r.Get("/getReview", s.userHandler.GetUserReview)
	})

	router.Route("/team", func(r chi.Router) {
		r.Post("/add", s.teamHandler.CreateTeam)
		r.Get("/get", s.teamHandler.GetTeam)
		r.Post("/deactivate", s.teamHandler.MassDeactivateTeamUsers)
	})

	router.Route("/pullRequest", func(r chi.Router) {
		r.Post("/create", s.prHandler.CreatePR)
		r.Post("/merge", s.prHandler.MergePR)
		r.Post("/reassign", s.prHandler.ReassignPR)
	})

	router.Route("/statistics", func(r chi.Router) {
		r.Get("/team", s.statsHandler.GetTeamStats)
		r.Get("/team/users", s.statsHandler.GetUserStats)
	})

	s.logger.Info("HTTP routes registered successfully")
}

func (s *Server) Stop(ctx context.Context, timeout time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server == nil || !s.isRunning {
		return nil
	}

	s.logger.Info("Shutting down HTTP server...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.logger.Error(fmt.Sprintf("Failed to shutdown server gracefully: %v", err))
		return err
	}

	s.isRunning = false
	s.logger.Info("HTTP server stopped successfully")
	return nil
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		_ = duration.Round(time.Millisecond)
		s.logger.Info("HTTP", "method",
			r.Method, "urlpath", r.URL.Path, "status", ww.Status(), "duration", duration,
			"remoteAddr", r.RemoteAddr)
	})
}
