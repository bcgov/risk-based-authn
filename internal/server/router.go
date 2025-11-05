package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"rba/internal/server/ruleRouter"
	"rba/util"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	fmt.Printf("%+v\n", s.rules)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Group for routes requiring auth
	r.Group(func(protected chi.Router) {
		protected.Use(AuthMiddleware(s.authKeys))
		protected.Post("/event", s.EventHandler)

		protected.Mount("/configuration/rules/denylist", ruleRouter.DenyListRouter(s.rules))
	})

	return r
}

type EventRequest struct {
	Event string                 `json:"event"`
	Data  map[string]interface{} `json:"data"`
}

func (s *Server) EventHandler(w http.ResponseWriter, r *http.Request) {
	var req EventRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// If event name is invalid send a 400 since we don't know which risk modules to run
	validEvents := map[string]bool{
		"login": true,
	}

	if !validEvents[req.Event] {
		http.Error(w, "Invalid event type", http.StatusBadRequest)
		return
	}

	riskHandlers, found := s.riskHandlers[req.Event]
	if !found || len(riskHandlers) == 0 {
		http.Error(w, "No handlers for event", http.StatusNotFound)
		return
	}

	riskAssessments := make(chan util.RiskResult, len(riskHandlers))
	var wg sync.WaitGroup

	for _, namedHandler := range riskHandlers {
		wg.Add(1)
		go func(h util.RiskHandlerFunc) {
			defer wg.Done()

			// Create a context with 100ms timeout
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			resultChan := make(chan util.RiskResult, 1)

			// Need to run this in a routine so it does not block the context check for timeout
			go func() {
				resultChan <- namedHandler.Handler(ctx, req.Data)
			}()

			select {
			case <-ctx.Done():
				// If handler takes too long send back an error for the result
				errText := "deadline exceeded"
				riskAssessments <- util.RiskResult{
					Name:     namedHandler.Name,
					Score:    0,
					Err:      &errText,
					Strategy: namedHandler.Strategy,
				}
			case result := <-resultChan:
				// Otherwise include the handler result
				riskAssessments <- result
			}
		}(namedHandler.Handler)
	}

	go func() {
		wg.Wait()
		close(riskAssessments)
	}()

	type RiskResponse struct {
		Risk        float64           `json:"risk"`
		RuleResults []util.RiskResult `json:"ruleResults"`
	}

	avg, results := util.CalculateRisk(riskAssessments)

	if s.services.Nats.Enabled && avg > float64(s.services.Nats.Threshold) {
		util.PublishMessage(results)
	}

	response := RiskResponse{
		Risk:        avg,
		RuleResults: results,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
