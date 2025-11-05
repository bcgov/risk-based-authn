package ruleRouter

import (
	"encoding/json"
	"log"
	"net/http"
	"rba/rules"
	"rba/types"

	"github.com/go-chi/chi/v5"
)

type DenylistGetResponse struct {
	CIDRs []string `json:"cidrs"`
	IPs   []string `json:"ips"`
}

func DenyListRouter(rulesConfig []types.RuleConfig) chi.Router {

	router := chi.NewRouter()

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		cidrs, err := rules.GetDenylistParams(rulesConfig, "cidrs")
		if err != nil {
			log.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ips, err := rules.GetDenylistParams(rulesConfig, "ips")
		if err != nil {
			log.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		response := DenylistGetResponse{
			CIDRs: cidrs,
			IPs:   ips,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	router.Put("/", func(w http.ResponseWriter, r *http.Request) {
		type DenylistUpdate struct {
			ParamType string `json:"type"`
			Value     string `json:"value"`
		}

		var payload DenylistUpdate
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}

		log.Print(payload)
		defer r.Body.Close()

		err := rules.UpdateDenylistParam(rulesConfig, payload.Value, payload.ParamType, "add")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	})

	return router
}
