package presence

import "strings"

// buildPrettySpotify mirrors zRuvix.Presence.Spotify.build_pretty_spotify.
// It returns nil when the activity is nil.
func buildPrettySpotify(activity map[string]any) *Spotify {
	if activity == nil {
		return nil
	}
	return &Spotify{
		TrackID:     spotifyTrackID(activity),
		Artist:      activity["state"],
		Song:        activity["details"],
		Album:       spotifyAlbumTitle(activity),
		AlbumArtURL: spotifyAlbumArtURL(activity),
		Timestamps:  activity["timestamps"],
	}
}

func spotifyTrackID(activity map[string]any) *string {
	if id, ok := activity["sync_id"].(string); ok {
		return &id
	}
	return nil
}

// spotifyAlbumTitle reads assets.large_text (or a bare string assets value),
// matching the get_album_title clauses.
func spotifyAlbumTitle(activity map[string]any) *string {
	switch assets := activity["assets"].(type) {
	case map[string]any:
		if lt, ok := assets["large_text"].(string); ok {
			return &lt
		}
	case string:
		return &assets
	}
	return nil
}

// spotifyAlbumArtURL resolves assets.large_image (form "spotify:<id>") into a
// scdn image URL, matching get_album_art_url.
func spotifyAlbumArtURL(activity map[string]any) *string {
	assets, ok := activity["assets"].(map[string]any)
	if !ok {
		return nil
	}
	large, ok := assets["large_image"].(string)
	if !ok {
		return nil
	}
	parts := strings.SplitN(large, ":", 2)
	if len(parts) == 2 {
		url := "https://i.scdn.co/image/" + parts[1]
		return &url
	}
	return nil
}
