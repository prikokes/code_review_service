package app

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"CodeRewievService/internal/database"
	httpInterface "CodeRewievService/internal/http"
	"CodeRewievService/internal/interfaces"
)

type App struct {
	wg         sync.WaitGroup
	ctx        context.Context
	container  Container
	mu         sync.Mutex
	isRunning  bool
	cancel     context.CancelFunc
	logger     *slog.Logger
	shutdownCh chan os.Signal
}

func NewApp() *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		ctx:    ctx,
		wg:     sync.WaitGroup{},
		logger: slog.Default(),
		cancel: cancel,
	}
}

type Container struct {
	server interfaces.Server
}

func (a *App) Start() error {
	a.logger.Info("Starting application...")
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := a.container.server.Start(a.ctx); err != nil {
			a.logger.Error("Failed to start the server")
		}
	}()

	a.mu.Lock()
	a.isRunning = true
	a.mu.Unlock()

	return nil
}

func (a *App) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.isRunning {
		a.logger.Debug("Application is not running, nothing to stop")
		return nil
	}

	a.logger.Info("Stopping application...")
	a.logger.Debug("Cancelling context")

	// Cancel main context to signal all goroutines to stop
	a.cancel()

	a.logger.Info("Stopping app components...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.container.server.Stop(shutdownCtx, 10*time.Second); err != nil {
		a.logger.Error(fmt.Sprintf("Failed to stop server: %v", err))
	}

	// Wait for goroutines to finish with timeout
	a.logger.Debug("Waiting for goroutines to finish")
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		a.logger.Debug("All goroutines finished")
	case <-time.After(5 * time.Second):
		a.logger.Warn("Stop timeout exceeded, forcing stop")
	}
	a.isRunning = false
	a.logger.Info("Application stopped")
	a.logger.Debug("Application stopped successfully")

	close(a.shutdownCh)

	return nil
}

func (a *App) WaitForShutdown() error {
	select {
	case <-a.ctx.Done():
		a.logger.Info("Application stopped before shutdown")
		return fmt.Errorf("application stopped before shutdown")
	case <-a.shutdownCh:
		a.logger.Info("Initializing graceful shutdown")
		err := a.Stop()
		if err != nil {
			a.logger.Error("Error during graceful shutdown: ", err)
			return err
		}
		return nil
	}
}

func (a *App) Run() error {
	db := database.InitDB()

	err := godotenv.Load()

	address := os.Getenv("APP_ADDRESS")
	port, err := strconv.Atoi(os.Getenv("APP_PORT"))

	if err != nil {
		port = 0
	}

	a.container.server = httpInterface.NewServer(a.logger, db, address, port)

	err = a.Start()
	if err != nil {
		a.logger.Error(fmt.Sprintf("Cannot start application %v", err))
		return err
	}

	a.logger.Info("Application started")

	a.shutdownCh = make(chan os.Signal, 1)
	signal.Notify(a.shutdownCh, os.Interrupt, syscall.SIGTERM)

	err = a.WaitForShutdown()
	if err != nil {
		a.logger.Error(fmt.Sprintf("Error occured while waiting for shutdown %v", err))
		return err
	}

	a.logger.Info("Application shutdown completed successfully")
	return nil
}
