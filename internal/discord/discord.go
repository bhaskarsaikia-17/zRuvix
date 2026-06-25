// Package discord is a tiny Discord REST helper used to fetch data that the
// gateway does not include in its payloads (e.g. a user's banner and accent
// color, which are only present on the full GET /users/{id} object). It is kept
// dependency-light (config + net/http) so it can be imported by presence
// without creating an import cycle.
package discord

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"zruvix/internal/config"
)

const apiHost = "https://discord.com/api/v9"

var client = &http.Client{Timeout: 10 * time.Second}

// FetchUser returns the full user object for id via GET /users/{id}, using the
// bot token. It is the only way to obtain banner / accent_color, which the
// gateway omits. Returns an error when no bot token is configured or the
// request fails.
func FetchUser(id string) (map[string]any, error) {
	if config.C == nil || config.C.BotToken == "" {
		return nil, fmt.Errorf("no bot token configured")
	}

	req, err := http.NewRequest(http.MethodGet, apiHost+"/users/"+id, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bot "+config.C.BotToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("discord: GET /users/%s returned %d", id, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}
