package presence

// SocketMessage is the envelope sent to client WebSocket connections. It maps
// to the maps Elixir builds, e.g. %{op: 0, t: "PRESENCE_UPDATE", d: ...}.
type SocketMessage struct {
	Op  int    `json:"op"`
	Seq *int   `json:"seq,omitempty"`
	T   string `json:"t,omitempty"`
	D   any    `json:"d,omitempty"`
}

// Subscriber is anything that can receive presence events — in practice a
// client WebSocket connection. Concrete implementations must be comparable
// (pointer types) so they can be used as map keys.
type Subscriber interface {
	SendEvent(SocketMessage)
}

// PrettyPresence is the user-facing presence object returned by the REST API
// and emitted over the socket. JSON tags match Lanyard.Presence.PrettyPresence.
type PrettyPresence struct {
	DiscordUser             map[string]any    `json:"discord_user"`
	DiscordStatus           string            `json:"discord_status"`
	ActiveOnDiscordWeb      bool              `json:"active_on_discord_web"`
	ActiveOnDiscordDesktop  bool              `json:"active_on_discord_desktop"`
	ActiveOnDiscordMobile   bool              `json:"active_on_discord_mobile"`
	ActiveOnDiscordEmbedded bool              `json:"active_on_discord_embedded"`
	ActiveOnDiscordVR       bool              `json:"active_on_discord_vr"`
	ListeningToSpotify      bool              `json:"listening_to_spotify"`
	Spotify                 *Spotify          `json:"spotify"`
	Activities              []any             `json:"activities"`
	KV                      map[string]string `json:"kv"`
}

// Spotify is the crafted spotify object, nil when the user is not listening.
type Spotify struct {
	TrackID     *string `json:"track_id"`
	Artist      any     `json:"artist"`
	Song        any     `json:"song"`
	Album       *string `json:"album"`
	AlbumArtURL *string `json:"album_art_url"`
	Timestamps  any     `json:"timestamps"`
}

// Error is a typed error carrying the HTTP status and machine code expected by
// the REST response helpers, mirroring Elixir's {:error, http, code, reason}.
type Error struct {
	HTTPCode int
	Code     string
	Message  string
}

func (e *Error) Error() string { return e.Message }
