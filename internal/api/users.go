package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"zruvix/internal/kv"
	"zruvix/internal/presence"
	"zruvix/internal/redis"
)

// usersRouter builds the /v1/users subtree.
func usersRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/@me", handleMe)
	r.Get("/{id}", handleGetUser)
	r.Patch("/{id}/kv", handlePatchKV)
	r.Put("/{id}/kv/{field}", handlePutKV)
	r.Delete("/{id}/kv/{field}", handleDeleteKV)
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) { notFound(w) })
	return r
}

// handleMe resolves the caller via their API key and returns their presence.
func handleMe(w http.ResponseWriter, r *http.Request) {
	key := authorizationHeader(r)
	userID := redis.Get("api_key:" + key)
	if userID == "" {
		noPermission(w)
		return
	}
	respondPresence(w, userID)
}

func handleGetUser(w http.ResponseWriter, r *http.Request) {
	respondPresence(w, chi.URLParam(r, "id"))
}

func respondPresence(w http.ResponseWriter, userID string) {
	p, err := presence.GetPrettyPresence(userID)
	if err != nil {
		respondError(w, err.HTTPCode, err.Code, err.Message)
		return
	}
	respondOK(w, p)
}

// handlePatchKV merges a JSON object of key/value pairs into the user's KV.
func handlePatchKV(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if !validateResourceAccess(r, userID) {
		noPermission(w)
		return
	}

	body, _ := io.ReadAll(r.Body)
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		respondError(w, http.StatusNotFound, "invalid_kv_value", "body must be an object")
		return
	}

	pairs := make(map[string]string, len(parsed))
	for k, v := range parsed {
		s, ok := v.(string)
		if !ok {
			// Non-string value: matches the Elixir rescue path.
			respondError(w, http.StatusNotFound, "invalid_kv_value", "body must be an object")
			return
		}
		if verr := kv.ValidatePair(k, s); verr != nil {
			respondError(w, http.StatusNotFound, "kv_validation_failed", verr.Error())
			return
		}
		pairs[k] = s
	}

	if err := kv.Multiset(userID, pairs); err != nil {
		respondError(w, http.StatusNotFound, "kv_validation_failed", err.Error())
		return
	}
	respondNoContent(w)
}

// handlePutKV sets a single key to the request body.
func handlePutKV(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	field := chi.URLParam(r, "field")
	if !validateResourceAccess(r, userID) {
		noPermission(w)
		return
	}

	body, _ := io.ReadAll(r.Body)
	if _, err := kv.Set(userID, field, string(body)); err != nil {
		respondError(w, http.StatusNotFound, "kv_validation_failed", err.Error())
		return
	}
	respondNoContent(w)
}

// handleDeleteKV removes a single key.
func handleDeleteKV(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	field := chi.URLParam(r, "field")
	if !validateResourceAccess(r, userID) {
		noPermission(w)
		return
	}
	_ = kv.Del(userID, field)
	respondNoContent(w)
}

// validateResourceAccess checks the Authorization header maps to userID.
func validateResourceAccess(r *http.Request, userID string) bool {
	key := authorizationHeader(r)
	return redis.Get("api_key:"+key) == userID
}
