// Package main is the entry point for the Custodia password management server.
// It loads configuration, initializes logging, sets up database storage,
// creates the gRPC server with TLS support (optional), and handles graceful
// shutdown on system signals.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/max-marek-projects/custodia/internal/config"
	"github.com/max-marek-projects/custodia/internal/handlers"
	"github.com/max-marek-projects/custodia/internal/logger"
	"github.com/max-marek-projects/custodia/internal/repository"
	"github.com/max-marek-projects/custodia/internal/server"
	"github.com/max-marek-projects/custodia/internal/service"
	"github.com/max-marek-projects/custodia/internal/utils"
)

// global variables that can be rewritten by flags
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

// run is the server entry point.
// It performs the following steps:
//  1. Loads configuration from flags, environment, and config file.
//  2. Initializes the logger.
//  3. Creates the database storage (runs migrations).
//  4. Initializes the business logic service.
//  5. Sets up the gRPC handler and server (with optional TLS).
//  6. Starts the gRPC server in a separate goroutine.
//  7. Waits for a termination signal (SIGINT, SIGTERM, SIGQUIT) or a server error.
//  8. On shutdown, gracefully stops the gRPC server with a 5-second timeout.
func run() error {
	fmt.Println("Build version:", utils.OrNA(buildVersion))
	fmt.Println("Build date:", utils.OrNA(buildDate))
	fmt.Println("Build commit:", utils.OrNA(buildCommit))
	// Load configuration.
	configData, err := config.LoadConfig()
	if err != nil {
		return err
	}
	// Initialize the global logger.
	err = logger.Initialize(configData.LoggerLevel)
	if err != nil {
		return err
	}
	// Create database storage (runs migrations).
	store, err := repository.NewDBStorage(configData.DatabaseURI, configData.ForceMigrations)
	if err != nil {
		logger.Log.Error("Unable to create storage", slog.Any("error", err))
		return err
	}
	// Create the business logic service.
	service, err := service.NewService(
		store,
		configData.CookieSecret,
		configData.AccessTokenLifespan.Duration(),
		configData.RefreshTokenLifespan.Duration(),
	)
	if err != nil {
		logger.Log.Error("Unable to create service handler", slog.Any("error", err))
		return err
	}
	grpcHandler := handlers.NewGRPCHandler(service)

	// Prepare TLS configuration if HTTPS is enabled.
	tlsConfig, err := server.CreateTLSConf(configData.CertPath, configData.KeyPath)
	if err != nil {
		logger.Log.Error("failed to load TLS certificates", slog.Any("error", err))
		return err
	}
	// Create the gRPC server.
	grpcSrv, err := server.NewServer(
		configData.RunAddr,
		grpcHandler,
		configData.ReadTimeout.Duration(),
		configData.WriteTimeout.Duration(),
		configData.CookieSecret,
		tlsConfig,
	)
	if err != nil {
		logger.Log.Error("Unable to initialize server", slog.Any("error", err))
		return err
	}

	// Run the gRPC server in a separate goroutine.
	grpcErr := make(chan error, 1)
	go func() {
		grpcErr <- grpcSrv.ListenAndServe()
	}()

	// Set up signal handling for graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(stop)

	// Wait for either a shutdown signal or a server error.
	select {
	case sig := <-stop:
		logger.Log.Info("Shutdown signal received", slog.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Gracefully stop the gRPC server.
		if err := grpcSrv.Shutdown(ctx); err != nil {
			logger.Log.Error("gRPC graceful shutdown failed", slog.Any("error", err))
			return err
		} else {
			logger.Log.Info("gRPC server stopped gracefully")
		}

	case err := <-grpcErr:
		if err != nil {
			logger.Log.Error("gRPC server stopped with error", slog.Any("error", err))
			return err
		}
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
