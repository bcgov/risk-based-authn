package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"rba/rules"
	"rba/util"
	"testing"
)

func TestHandler(t *testing.T) {
	newServer := &Server{
		port:         8080,
		riskHandlers: map[string][]util.NamedRiskHandler{},
		services:     rules.ServicesConfig{},
	}

	// Create an httptest server from your handler
	ts := httptest.NewServer(newServer.RegisterRoutes())
	defer ts.Close()

	// Make a request to the server’s base URL
	resp, err := http.Get(fmt.Sprintf("%s/health", ts.URL))
	if err != nil {
		t.Fatalf("error making request to server: %v", err)
	}
	defer resp.Body.Close()

	// Assertions
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", resp.Status)
	}
}
