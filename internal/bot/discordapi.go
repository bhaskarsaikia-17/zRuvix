// Package bot implements the Discord bot: the REST API used to reply to users,
// the command handler, and the individual KV commands.
package bot

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"zruvix/internal/config"
	"zruvix/internal/metrics"
)

const apiHost = "https://discord.com/api/v9"

var httpClient = &http.Client{}

func authHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bot "+config.C.BotToken)
	req.Header.Set("Content-Type", "application/json")
}

// SendMessage posts a plain-text message to a channel. Bare "@" is broken with
// a zero-width space so the bot can't be used to mass-mention.
func SendMessage(channelID, content string) {
	metrics.DiscordMessagesSent.Inc()
	sanitized := strings.ReplaceAll(content, "@", "@\u200b")
	body, _ := json.Marshal(map[string]any{"content": sanitized})
	post(apiHost+"/channels/"+channelID+"/messages", body)
}

// SendEmbed posts a single embed to a channel.
func SendEmbed(channelID string, embed map[string]any) {
	metrics.DiscordMessagesSent.Inc()
	body, _ := json.Marshal(map[string]any{"embeds": []any{embed}})
	post(apiHost+"/channels/"+channelID+"/messages", body)
}

// CreateDM opens (or fetches) a DM channel with a user and returns its id.
func CreateDM(recipientID string) string {
	body, _ := json.Marshal(map[string]any{"recipient_id": recipientID})
	resp := post(apiHost+"/users/@me/channels", body)
	if resp == nil {
		return ""
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var parsed map[string]any
	if json.Unmarshal(data, &parsed) == nil {
		if id, ok := parsed["id"].(string); ok {
			return id
		}
	}
	return ""
}

func post(url string, body []byte) *http.Response {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil
	}
	authHeaders(req)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil
	}
	// Callers that don't read the body still need it drained/closed.
	if url != apiHost+"/users/@me/channels" {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	return resp
}
