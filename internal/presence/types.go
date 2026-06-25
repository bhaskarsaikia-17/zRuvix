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
// and emitted over the socket. JSON tags match zRuvix.Presence.PrettyPresence.
type PrettyPresence struct {
	DiscordUser             map[string]any    `json:"discord_user"`
	DiscordStatus           string            `json:"discord_status"`
	ActiveOnDiscordWeb      bool              `json:"active_on_discord_web"`
	ActiveOnDiscordDesktop  bool              `json:"active_on_discord_desktop"`
	ActiveOnDiscordMobile   bool              `json:"active_on_discord_mobile"`
	ActiveOnDiscordEmbedded bool              `json:"active_on_discord_embedded"`
	ActiveOnDiscordVR       bool              `json:"active_on_discord_vr"`
	CustomStatus            *CustomStatus     `json:"custom_status"`
	ListeningToSpotify      bool              `json:"listening_to_spotify"`
	Spotify                 *Spotify          `json:"spotify"`
	ListeningToYouTubeMusic bool              `json:"listening_to_youtube_music"`
	YouTubeMusic            *YouTubeMusic     `json:"youtube_music"`
	NowPlaying              *NowPlaying       `json:"now_playing"`
	Activities              []any             `json:"activities"`
	Banner                  *string           `json:"banner"`
	BannerURL               *string           `json:"banner_url"`
	AccentColor             *int              `json:"accent_color"`
	MemberSince             *int64            `json:"member_since"`
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

// YouTubeMusic is the crafted YouTube Music object, nil when not listening.
// It mirrors the Spotify object's shape, plus a url to the track.
type YouTubeMusic struct {
	TrackID     *string `json:"track_id"`      // YouTube video id, parsed from the track url
	Artist      any     `json:"artist"`        // activity "state"
	Song        any     `json:"song"`          // activity "details"
	Album       *string `json:"album"`         // assets.large_text
	AlbumArtURL *string `json:"album_art_url"` // resolved assets.large_image
	URL         *string `json:"url"`           // link to the track (details_url)
	Timestamps  any     `json:"timestamps"`
}

// CustomStatus is the user's Discord custom status (the activity with type 4),
// surfaced as a clean field. It is nil when the user has no custom status set.
// The raw activity also remains in the Activities array.
type CustomStatus struct {
	Text  *string            `json:"text"`  // the status text (activity "state"), nil when only an emoji is set
	Emoji *CustomStatusEmoji `json:"emoji"` // nil when no emoji is set
}

// CustomStatusEmoji describes the emoji attached to a custom status. For a
// unicode emoji, Name is the character and ID is nil; for a custom guild emoji,
// Name is the emoji name and ID is its snowflake.
type CustomStatusEmoji struct {
	Name     *string `json:"name"`
	ID       *string `json:"id"`
	Animated bool    `json:"animated"`
}

// NowPlaying is a single, source-agnostic music object that normalizes whatever
// the user is currently listening to (Spotify, YouTube Music, ...). It is nil
// when nothing is playing. Clients can render one consistent widget regardless
// of source, and compute live progress from timestamps + duration_ms.
type NowPlaying struct {
	Source      string  `json:"source"` // "spotify" | "youtube_music"
	Song        any     `json:"song"`
	Artist      any     `json:"artist"`
	Album       *string `json:"album"`
	AlbumArtURL *string `json:"album_art_url"`
	TrackID     *string `json:"track_id"`
	TrackURL    *string `json:"track_url"`
	Timestamps  any     `json:"timestamps"`  // {start, end} in ms, when known
	DurationMs  *int64  `json:"duration_ms"` // end-start, when both known
}

// Error is a typed error carrying the HTTP status and machine code expected by
// the REST response helpers, mirroring Elixir's {:error, http, code, reason}.
type Error struct {
	HTTPCode int
	Code     string
	Message  string
}

func (e *Error) Error() string { return e.Message }
