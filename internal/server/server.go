package server

import (
	"fmt"
	"net/http"
	"os"
	"rba/rules"
	"rba/util"
	"strconv"
	"time"
)

type Server struct {
	port         int
	riskHandlers map[string][]util.NamedRiskHandler
	cfg          rules.Config
}

func NewServer(riskHandlers map[string][]util.NamedRiskHandler, cfg rules.Config) *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))

	NewServer := &Server{
		port:         port,
		riskHandlers: riskHandlers,
		cfg:          cfg,
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
