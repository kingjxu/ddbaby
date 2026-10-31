// Command gto-solver is a drop-in replacement for the upstream /gto/decision
// and /gto/decision_v2 endpoints. It keeps the existing request/response
// contract and routes each request through preflop tables and a heuristic
// fallback (optionally an L2 solver sidecar).
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kingjxu/ddbaby/gto"
)

func main() {
	defaultAddr := env("GTO_SOLVER_ADDR", ":9200")
	defaultTables := env("GTO_PREFLOP_TABLES", "./data/preflop")
	defaultSolverURL := env("GTO_SOLVER_SIDECAR_URL", "")
	defaultSolverTimeout := envInt("GTO_SOLVER_TIMEOUT_MS", 1500)
	defaultDecisionTimeout := envInt("GTO_DECISION_TIMEOUT_MS", 4000)
	defaultSamples := envInt("GTO_MC_SAMPLES", 2000)
	defaultAPIKey := env("GTO_SOLVER_API_KEY", env("GTO_API_KEY", ""))

	addr := flag.String("addr", defaultAddr, "listen address")
	tablesDir := flag.String("tables", defaultTables, "directory with preflop-table/v1 JSON files")
	solverURL := flag.String("solver-url", defaultSolverURL, "optional L2 solver sidecar base URL")
	solverTimeout := flag.Int("solver-timeout-ms", defaultSolverTimeout, "L2 solver timeout in milliseconds")
	decisionTimeout := flag.Int("decision-timeout-ms", defaultDecisionTimeout, "total decision timeout in milliseconds")
	samples := flag.Int("mc-samples", defaultSamples, "Monte Carlo samples for the heuristic fallback")
	apiKey := flag.String("api-key", defaultAPIKey, "required X-API-Key value; empty disables auth")
	flag.Parse()

	logger := log.New(os.Stdout, "", log.LstdFlags|log.LUTC)
	service, err := gto.New(gto.Options{
		TablesDir:       *tablesDir,
		SolverURL:       *solverURL,
		SolverTimeout:   time.Duration(*solverTimeout) * time.Millisecond,
		DecisionTimeout: time.Duration(*decisionTimeout) * time.Millisecond,
		Samples:         *samples,
		Logger:          logger,
	})
	if err != nil {
		logger.Fatalf("[gto] init failed: %v", err)
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           gto.NewServer(service, *apiKey, logger).Handler(),
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Printf("[gto] listening on %s tables=%q solver=%t auth=%t",
			*addr, *tablesDir, *solverURL != "", *apiKey != "")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("[gto] listen failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Printf("[gto] shutdown error: %v", err)
	}
	logger.Printf("[gto] stopped")
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
