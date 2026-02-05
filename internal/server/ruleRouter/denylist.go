package ruleRouter

import (
	"encoding/json"
	"net/http"
	"net/url"
	"rba/rules"
	"rba/util"

	"github.com/go-chi/chi/v5"
)

type DenylistGetResponse struct {
	Networks []string `json:"networks"`
}

func DenyListRouter() chi.Router {

	router := chi.NewRouter()

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		networks, err := rules.Denylist.GetNetworks(r.Context())
		if err != nil {
			http.Error(w, err.Error(), util.HttpStatusCodeForError(err))
			return
		}

		w.Header().Set("Content-Type", "application/json")

		response := DenylistGetResponse{
			Networks: networks,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
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

		err := rules.Denylist.AddNetwork(r.Context(), payload.Value)
		if err != nil {
			http.Error(w, err.Error(), util.HttpStatusCodeForError(err))
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	router.Delete("/{entry}", func(w http.ResponseWriter, r *http.Request) {
		rawEntry := chi.URLParam(r, "entry")

		entry, err := url.PathUnescape(rawEntry)
		if err != nil {
			http.Error(w, "invalid encoding", http.StatusBadRequest)
			return
		}

		err = rules.Denylist.RemoveNetwork(r.Context(), entry)
		if err != nil {
			http.Error(w, err.Error(), util.HttpStatusCodeForError(err))
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	return router
}
