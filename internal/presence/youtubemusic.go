package presence

import (
	"net/url"
	"strings"
)

// buildPrettyYouTubeMusic crafts the youtube_music object from a YouTube Music
// rich-presence activity. Returns nil when the activity is nil.
//
// Example raw activity fields:
//
//	name:        "YouTube Music"
//	details:     "Reminder"            (song title)
//	state:       "The Weeknd"          (artist)
//	assets.large_text:  "Starboy"      (album)
//	assets.large_image: "mp:external/<hash>/https/<host>/<path>"  (Discord-proxied art)
//	details_url: "https://music.youtube.com/watch?v=a40tAP5MC6M"  (track link)
//	timestamps:  {start, end}
func buildPrettyYouTubeMusic(activity map[string]any) *YouTubeMusic {
	if activity == nil {
		return nil
	}
	trackURL := stringPtr(activity["details_url"])
	return &YouTubeMusic{
		TrackID:     youtubeTrackID(trackURL),
		Artist:      activity["state"],
		Song:        activity["details"],
		Album:       spotifyAlbumTitle(activity), // same assets.large_text logic
		AlbumArtURL: resolveActivityImage(activity),
		URL:         trackURL,
		Timestamps:  activity["timestamps"],
	}
}

// youtubeTrackID extracts the YouTube video id (?v=...) from the track url.
func youtubeTrackID(trackURL *string) *string {
	if trackURL == nil {
		return nil
	}
	u, err := url.Parse(*trackURL)
	if err != nil {
		return nil
	}
	if v := u.Query().Get("v"); v != "" {
		return &v
	}
	return nil
}

// resolveActivityImage turns Discord's assets.large_image into a usable URL.
// YouTube Music art arrives as an external-proxy reference ("mp:external/...");
// other apps may use http(s) URLs directly. Returns nil if unresolvable.
func resolveActivityImage(activity map[string]any) *string {
	assets, ok := activity["assets"].(map[string]any)
	if !ok {
		return nil
	}
	img, ok := assets["large_image"].(string)
	if !ok || img == "" {
		return nil
	}
	switch {
	case strings.HasPrefix(img, "mp:"):
		// Discord media-proxy reference -> served from media.discordapp.net.
		u := "https://media.discordapp.net/" + strings.TrimPrefix(img, "mp:")
		return &u
	case strings.HasPrefix(img, "http://"), strings.HasPrefix(img, "https://"):
		return &img
	default:
		return nil
	}
}

// stringPtr returns a *string if v is a non-empty string, else nil.
func stringPtr(v any) *string {
	if s, ok := v.(string); ok && s != "" {
		return &s
	}
	return nil
}
