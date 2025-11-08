package ruleRouter

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"rba/rules"

	"github.com/go-chi/chi/v5"
)

type DenylistGetResponse struct {
	CIDRs []string `json:"cidrs"`
	IPs   []string `json:"ips"`
}

func DenyListRouter() chi.Router {

	router := chi.NewRouter()

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		cidrs, errCode, err := rules.GetDenylistParams(r.Context(), "cidrs")
		if err != nil {
			log.Print(err)
			http.Error(w, err.Error(), errCode)
			return
		}

		ips, errCode, err := rules.GetDenylistParams(r.Context(), "ips")
		if err != nil {
			log.Print(err)
			http.Error(w, err.Error(), errCode)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		response := DenylistGetResponse{
			CIDRs: cidrs,
			IPs:   ips,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
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

		defer r.Body.Close()

		errCode, err := rules.UpdateDenylistParam(r.Context(), payload.Value, payload.ParamType, "add")
		if err != nil {
			http.Error(w, err.Error(), errCode)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	})

	router.Delete("/{paramType}/{entry}", func(w http.ResponseWriter, r *http.Request) {
		rawEntry := chi.URLParam(r, "entry")
		paramType := chi.URLParam(r, "paramType")

		entry, err := url.PathUnescape(rawEntry)
		if err != nil {
			http.Error(w, "invalid encoding", http.StatusBadRequest)
			return
		}

		errCode, err := rules.RemoveDenylistEntry(r.Context(), paramType, entry)
		if err != nil {
			http.Error(w, err.Error(), errCode)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	})

	return router
}
