// Package api implements the HTTP layer: routing, REST response helpers, the
// KV/user endpoints, and the Discord avatar quicklink proxy. It mirrors
// Lanyard.Api.Router and friends.
package api

import (
	"encoding/json"
	"net/http"
)

// respondOK writes {"success": true, "data": data} with status 200.
func respondOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

// respondNoContent writes a 204, matching Util.respond(conn, {:ok}).
func respondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// respondError writes {"success": false, "error": {code, message}}.
func respondError(w http.ResponseWriter, httpCode int, code, message string) {
	writeJSON(w, httpCode, map[string]any{
		"success": false,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func notFound(w http.ResponseWriter) {
	respondError(w, http.StatusNotFound, "not_found", "Route does not exist")
}

func noPermission(w http.ResponseWriter) {
	respondError(w, http.StatusUnauthorized, "no_permission", "You do not have permission to access this resource")
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func authorizationHeader(r *http.Request) string {
	return r.Header.Get("Authorization")
}
