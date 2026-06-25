package presence

import (
	"encoding/json"
	"strconv"
	"time"

	"zruvix/internal/redis"
)

// historyCap is the max number of events retained per list.
const historyCap = 100

func statusKey(uid string) string   { return "zruvix_hist_status:" + uid }
func tracksKey(uid string) string   { return "zruvix_hist_tracks:" + uid }
func lastSeenKey(uid string) string { return "zruvix_last_seen:" + uid }
func tracksTodayKey(uid string) string {
	return "zruvix_stat_tracks:" + uid + ":" + time.Now().UTC().Format("20060102")
}

// recordHistory diffs the freshly-built presence against the last seen state and
// appends status-change and track-change events to Redis. It also refreshes the
// last_seen timestamp and the daily track counter.
func recordHistory(p *Presence, pretty *PrettyPresence) {
	p.mu.Lock()
	newStatus := pretty.DiscordStatus
	newTrack := ""
	if pretty.NowPlaying != nil && pretty.NowPlaying.TrackID != nil {
		newTrack = *pretty.NowPlaying.TrackID
	}
	statusChanged := newStatus != p.lastStatus
	trackChanged := newTrack != "" && newTrack != p.lastTrackID
	p.lastStatus = newStatus
	if newTrack != "" {
		p.lastTrackID = newTrack
	}
	p.mu.Unlock()

	now := time.Now().UnixMilli()
	redis.Set(lastSeenKey(p.UserID), strconv.FormatInt(now, 10))

	if statusChanged {
		b, _ := json.Marshal(map[string]any{"status": newStatus, "ts": now})
		redis.LPush(statusKey(p.UserID), string(b))
		redis.LTrim(statusKey(p.UserID), 0, historyCap-1)
	}

	if trackChanged && pretty.NowPlaying != nil {
		np := pretty.NowPlaying
		b, _ := json.Marshal(map[string]any{
			"source":    np.Source,
			"song":      np.Song,
			"artist":    np.Artist,
			"album":     np.Album,
			"track_id":  np.TrackID,
			"track_url": np.TrackURL,
			"ts":        now,
		})
		redis.LPush(tracksKey(p.UserID), string(b))
		redis.LTrim(tracksKey(p.UserID), 0, historyCap-1)

		k := tracksTodayKey(p.UserID)
		redis.Incr(k)
		redis.Expire(k, 60*60*48) // keep daily counters ~2 days
	}
}

// History returns recent status-change and track-change events for a user.
// Works even when the user is offline/unmonitored (read straight from Redis).
func History(uid string) map[string]any {
	return map[string]any{
		"status_history": rawList(redis.LRange(statusKey(uid), 0, historyCap-1)),
		"track_history":  rawList(redis.LRange(tracksKey(uid), 0, historyCap-1)),
	}
}

// Stats returns aggregate presence statistics for a user.
func Stats(uid string) map[string]any {
	var lastSeen any
	if v := redis.Get(lastSeenKey(uid)); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			lastSeen = n
		}
	}

	tracksToday := 0
	if v := redis.Get(tracksTodayKey(uid)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			tracksToday = n
		}
	}

	// Current status: prefer the live registry, fall back to last recorded event.
	currentStatus := "offline"
	if p, ok := Reg.Lookup(uid); ok {
		currentStatus = p.BuildPretty().DiscordStatus
	} else if head := redis.LRange(statusKey(uid), 0, 0); len(head) > 0 {
		var e struct {
			Status string `json:"status"`
		}
		if json.Unmarshal([]byte(head[0]), &e) == nil && e.Status != "" {
			currentStatus = e.Status
		}
	}

	return map[string]any{
		"current_status":      currentStatus,
		"last_seen":           lastSeen,
		"tracks_today":        tracksToday,
		"is_monitored":        isMonitored(uid),
		"track_history_count": len(redis.LRange(tracksKey(uid), 0, historyCap-1)),
	}
}

func isMonitored(uid string) bool {
	_, ok := Reg.Lookup(uid)
	return ok
}

// rawList wraps stored JSON strings so they embed as objects in the response.
func rawList(items []string) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(items))
	for _, s := range items {
		out = append(out, json.RawMessage(s))
	}
	return out
}
