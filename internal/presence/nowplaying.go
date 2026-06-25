package presence

// buildNowPlaying normalizes the active music source into a single object.
// Spotify takes precedence when both are somehow present. Returns nil when
// nothing is playing.
func buildNowPlaying(spotify *Spotify, ytm *YouTubeMusic) *NowPlaying {
	switch {
	case spotify != nil:
		var trackURL *string
		if spotify.TrackID != nil {
			u := "https://open.spotify.com/track/" + *spotify.TrackID
			trackURL = &u
		}
		return &NowPlaying{
			Source:      "spotify",
			Song:        spotify.Song,
			Artist:      spotify.Artist,
			Album:       spotify.Album,
			AlbumArtURL: spotify.AlbumArtURL,
			TrackID:     spotify.TrackID,
			TrackURL:    trackURL,
			Timestamps:  spotify.Timestamps,
			DurationMs:  durationMs(spotify.Timestamps),
		}
	case ytm != nil:
		return &NowPlaying{
			Source:      "youtube_music",
			Song:        ytm.Song,
			Artist:      ytm.Artist,
			Album:       ytm.Album,
			AlbumArtURL: ytm.AlbumArtURL,
			TrackID:     ytm.TrackID,
			TrackURL:    ytm.URL,
			Timestamps:  ytm.Timestamps,
			DurationMs:  durationMs(ytm.Timestamps),
		}
	default:
		return nil
	}
}

// durationMs computes end-start (ms) from a timestamps map, when both are known.
func durationMs(ts any) *int64 {
	m, ok := ts.(map[string]any)
	if !ok {
		return nil
	}
	start, sok := toInt64(m["start"])
	end, eok := toInt64(m["end"])
	if !sok || !eok || end <= start {
		return nil
	}
	d := end - start
	return &d
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	default:
		return 0, false
	}
}
