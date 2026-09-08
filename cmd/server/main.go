// Package main is the entry point for the for a password management server.

package main

import (
	"context"
	"crypto/tls"
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
)

func main() {
	configData, err := config.LoadConfig()
	if err != nil {
		logger.Log.Error("failed to initialize settings", slog.Any("error", err))
		os.Exit(1)
	}
	err = logger.Initialize(configData.LoggerLevel)
	if err != nil {
		logger.Log.Error("failed to initialize logger", slog.Any("error", err))
		os.Exit(1)
	}
	store, err := repository.NewDBStorage(configData.DatabaseURI, configData.ForceMigrations)
	if err != nil {
		logger.Log.Error("Unable to create storage", slog.Any("error", err))
		os.Exit(1)
	}
	service := service.NewService(store, configData.CookieSecret, configData.AccessTokenLifespan.Duration(), configData.RefreshTokenLifespan.Duration())
	grpcHandler := handlers.NewGRPCHandler(service)

	// GRPC-server
	var tlsConfig *tls.Config
	if configData.EnableHTTPS {
		tlsConfig, err = server.CreateTLSConf("server.pem", "server.key")
		if err != nil {
			logger.Log.Error("failed to load TLS certificates", slog.Any("error", err))
			os.Exit(1)
		}
	}
	grpcSrv := server.NewServer(configData.RunAddr, grpcHandler, configData.ReadTimeout.Duration(), configData.WriteTimeout.Duration(), configData.CookieSecret, tlsConfig)

	// run gRPC in separate goroutine
	grpcErr := make(chan error, 1)
	go func() {
		grpcErr <- grpcSrv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(stop)

	// wait for shutdown signal or error
	select {
	case sig := <-stop:
		logger.Log.Info("Shutdown signal received", slog.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Graceful shutdown gRPC
		if err := grpcSrv.Shutdown(ctx); err != nil {
			logger.Log.Error("gRPC graceful shutdown failed", slog.Any("error", err))
		} else {
			logger.Log.Info("gRPC server stopped gracefully")
		}

	case err := <-grpcErr:
		if err != nil {
			logger.Log.Error("gRPC server stopped with error", slog.Any("error", err))
			os.Exit(1)
		}
	}
}
