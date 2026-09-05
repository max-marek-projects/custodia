// Package main is the entry point for the for a password management server.

package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
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
	configData := config.LoadConfig()
	err := logger.Initialize(configData.LoggerLevel)
	if err != nil {
		log.Fatalf("Unable to initialize logger: %v", err)
	}
	store, err := repository.NewDBStorage(configData.DatabaseURI, configData.ForceMigrations)
	if err != nil {
		logger.Log.Error("Unable to create storage", slog.Any("error", err))
		os.Exit(1)
	}
	service := service.NewService(store)
	httpHandler := handlers.NewHandler(service, configData.CookieSecret)
	// grpcHandler := handlers.NewGRPCHandler(service)
	// auditor, err := audit.InitAudit(configData.AuditFile, configData.AuditURL, configData.MaxParallelWorkers)
	// if err != nil {
	// 	logger.Log.Fatal("Unable to initialize audit", zap.Error(err))
	// }
	// defer auditor.Stop()

	// HTTP-server
	httpSrv := server.NewServer(configData.RunAddr, httpHandler, configData.ReadTimeout, configData.WriteTimeout, configData.CookieSecret)

	// gRPC-server
	// var tlsConfig *tls.Config
	// if configData.EnableHTTPS {
	// 	cert, err := tls.LoadX509KeyPair("server.pem", "server.key")
	// 	if err != nil {
	// 		logger.Log.Fatal("failed to load TLS certificates", zap.Error(err))
	// 	}
	// 	tlsConfig = &tls.Config{
	// 		Certificates: []tls.Certificate{cert},
	// 		MinVersion:   tls.VersionTLS12,
	// 	}
	// }
	// grpcSrv := server.NewGRPCServer(
	// 	configData.GRPCAddr,
	// 	grpcHandler,
	// 	configData.ReadTimeout,
	// 	configData.WriteTimeout,
	// 	auditor,
	// 	configData.CookieSecret,
	// 	tlsConfig,
	// )

	// pprof
	// go func() {
	// 	srv := &http.Server{
	// 		Addr:         "localhost:6060",
	// 		ReadTimeout:  5 * configData.ReadTimeout,
	// 		WriteTimeout: configData.WriteTimeout,
	// 		IdleTimeout:  120 * time.Second,
	// 	}
	// 	if err := srv.ListenAndServe(); err != nil {
	// 		logger.Log.Info("error running server", slog.Any("error", err))
	// 	}
	// 	defer srv.Shutdown(context.Background())
	// }()

	// run http in separate goroutine
	httpErr := make(chan error, 1)
	go func() {
		if configData.EnableHTTPS {
			httpErr <- httpSrv.ListenAndServeTLS("server.pem", "server.key")
		} else {
			httpErr <- httpSrv.ListenAndServe()
		}
	}()

	// run gRPC in separate goroutine
	// grpcErr := make(chan error, 1)
	// go func() {
	// 	grpcErr <- grpcSrv.ListenAndServe()
	// }()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(stop)

	// wait for shutdown signal or error
	select {
	case sig := <-stop:
		logger.Log.Info("Shutdown signal received", slog.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Graceful shutdown HTTP
		if err := httpSrv.Shutdown(ctx); err != nil {
			logger.Log.Error("HTTP graceful shutdown failed", slog.Any("error", err))
		} else {
			logger.Log.Info("HTTP server stopped gracefully")
		}

		// Graceful shutdown gRPC
		// if err := grpcSrv.Shutdown(ctx); err != nil {
		// 	logger.Log.Error("gRPC graceful shutdown failed", slog.Any("error", err))
		// } else {
		// 	logger.Log.Info("gRPC server stopped gracefully")
		// }

	case err := <-httpErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("HTTP server stopped with error", slog.Any("error", err))
			os.Exit(1)
		}
		// case err := <-grpcErr:
		// 	if err != nil {
		// 		logger.Log.Error("gRPC server stopped with error", slog.Any("error", err))
		// 		os.Exit(1)
		// 	}
	}
}
