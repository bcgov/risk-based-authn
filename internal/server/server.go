package server

import (
	"fmt"
	"net/http"
	"os"
	"rba/rules"
	"rba/types"
	"rba/util"
	"strconv"
	"time"
)

type Server struct {
	port         int
	riskHandlers map[string][]util.NamedRiskHandler
	services     rules.ServicesConfig
	rules        []types.RuleConfig
	authKeys     map[string][]byte
}

func NewServer(riskHandlers map[string][]util.NamedRiskHandler, services rules.ServicesConfig, rules []types.RuleConfig, authKeys map[string][]byte) *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))

	NewServer := &Server{
		port:         port,
		riskHandlers: riskHandlers,
		services:     services,
		rules:        rules,
		authKeys:     authKeys,
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
