// Package presence is the heart of Lanyard. It maintains one Presence per
// monitored user (the Go analogue of the GenRegistry-of-Presence GenServers),
// builds the user-facing PrettyPresence, caches it, and fans presence updates
// out to subscribed client sockets.
package presence

import (
	"encoding/json"
	"fmt"
	"sync"

	"zruvix/internal/config"
	"zruvix/internal/redis"
)

// Presence holds the live state for a single monitored user.
type Presence struct {
	mu              sync.RWMutex
	UserID          string
	DiscordUser     map[string]any
	DiscordPresence map[string]any
	KV              map[string]string
	subscribers     map[Subscriber]struct{}
}

// Registry is the concurrent set of monitored presences, replacing the Elixir
// GenRegistry of Lanyard.Presence workers.
type Registry struct {
	mu sync.RWMutex
	m  map[string]*Presence
}

// Reg is the process-wide presence registry.
var Reg = &Registry{m: make(map[string]*Presence)}

// presence cache (ETS :cached_presences equivalent), keyed by user id.
var (
	cacheMu sync.RWMutex
	cache   = make(map[string]PrettyPresence)
)

// global subscribers (ETS :global_subscribers equivalent) — sockets that asked
// to subscribe_to_all.
var (
	globalMu   sync.RWMutex
	globalSubs = make(map[Subscriber]struct{})
)

// Lookup returns the presence for id, if monitored.
func (r *Registry) Lookup(id string) (*Presence, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.m[id]
	return p, ok
}

// Count returns the number of monitored users.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.m)
}

// IDs returns all currently monitored user ids.
func (r *Registry) IDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.m))
	for id := range r.m {
		ids = append(ids, id)
	}
	return ids
}

// LookupOrStart returns the existing presence for id or creates a new one. On
// creation it loads the user's KV from Redis and broadcasts a PRESENCE_UPDATE
// to all global subscribers — matching Lanyard.Presence.init.
func (r *Registry) LookupOrStart(id string, discordPresence, discordUser map[string]any) *Presence {
	r.mu.Lock()
	if p, ok := r.m[id]; ok {
		r.mu.Unlock()
		return p
	}
	p := &Presence{
		UserID:          id,
		DiscordPresence: discordPresence,
		DiscordUser:     discordUser,
		subscribers:     make(map[Subscriber]struct{}),
	}
	r.m[id] = p
	r.mu.Unlock()

	// init: load KV, build (and cache) pretty presence, notify global subscribers.
	p.mu.Lock()
	p.KV = redis.HGetAll("lanyard_kv:" + id)
	p.mu.Unlock()

	pretty := p.BuildPretty()
	for _, sub := range globalSubscribers() {
		sub.SendEvent(SocketMessage{Op: 0, T: "PRESENCE_UPDATE", D: pretty})
	}
	return p
}

// Stop removes a presence from monitoring and drops its cache entry.
func (r *Registry) Stop(id string) {
	r.mu.Lock()
	delete(r.m, id)
	r.mu.Unlock()
	cacheDelete(id)
}

// SyncLocal merges diff into the presence and fans out to subscribers WITHOUT
// republishing on global_sync. This matches the Elixir gateway handlers which
// use GenServer.cast({:sync, ...}) directly (presence/user updates arrive from
// Discord on every node independently). Only KV changes go through Sync.
func (r *Registry) SyncLocal(userID string, diff map[string]any) {
	if p, ok := r.Lookup(userID); ok {
		p.applySync(diff)
	}
}

// Sync merges diff into the presence's state, rebuilds the pretty presence and
// fans it out to subscribers. Unless fromGlobal is set, it also republishes the
// diff on lanyard:global_sync for other nodes. Mirrors Lanyard.Presence.sync.
func (r *Registry) Sync(userID string, diff map[string]any, fromGlobal bool) {
	p, ok := r.Lookup(userID)
	if !ok {
		return
	}
	p.applySync(diff)

	if !fromGlobal {
		payload, _ := json.Marshal(map[string]any{
			"node_id": redis.NodeID,
			"user_id": userID,
			"diff":    diff,
		})
		redis.Publish("lanyard:global_sync", string(payload))
	}
}

func (p *Presence) applySync(diff map[string]any) {
	p.mu.Lock()
	if v, ok := diff["discord_presence"]; ok {
		p.DiscordPresence = asStringMap(v)
	}
	if v, ok := diff["discord_user"]; ok {
		p.DiscordUser = asStringMap(v)
	}
	if v, ok := diff["kv"]; ok {
		p.KV = asKV(v)
	}
	duser, dpres, kv := p.DiscordUser, p.DiscordPresence, p.KV
	subs := p.subscriberList()
	p.mu.Unlock()

	pretty := buildPretty(p.UserID, duser, dpres, kv)
	for _, s := range subs {
		s.SendEvent(SocketMessage{Op: 0, T: "PRESENCE_UPDATE", D: pretty})
	}
}

// BuildPretty builds (and caches) this presence's PrettyPresence.
func (p *Presence) BuildPretty() PrettyPresence {
	p.mu.RLock()
	duser, dpres, kv := p.DiscordUser, p.DiscordPresence, p.KV
	p.mu.RUnlock()
	return buildPretty(p.UserID, duser, dpres, kv)
}

// buildPretty mirrors Lanyard.Presence.build_pretty_presence and stores the
// result in the cache.
func buildPretty(userID string, discordUser, discordPresence map[string]any, kv map[string]string) PrettyPresence {
	if kv == nil {
		kv = map[string]string{}
	}

	var pretty PrettyPresence
	if discordPresence != nil {
		activities := discordPresence["activities"]
		var spotifyActivity map[string]any
		if list, ok := activities.([]any); ok {
			for _, a := range list {
				if m, ok := a.(map[string]any); ok {
					if id, _ := m["id"].(string); id == config.C.DiscordSpotifyActivity {
						spotifyActivity = m
						break
					}
				}
			}
		}

		status, _ := discordPresence["status"].(string)
		pretty = PrettyPresence{
			DiscordUser:             discordUser,
			DiscordStatus:           status,
			ActiveOnDiscordWeb:      clientStatusHas(discordPresence, "web"),
			ActiveOnDiscordDesktop:  clientStatusHas(discordPresence, "desktop"),
			ActiveOnDiscordMobile:   clientStatusHas(discordPresence, "mobile"),
			ActiveOnDiscordEmbedded: clientStatusHas(discordPresence, "embedded"),
			ActiveOnDiscordVR:       clientStatusHas(discordPresence, "vr"),
			ListeningToSpotify:      spotifyActivity != nil,
			Spotify:                 buildPrettySpotify(spotifyActivity),
			Activities:              buildPrettyActivities(activities),
			KV:                      kv,
		}
	} else {
		// No presence: defaults mirror the PrettyPresence struct defaults.
		pretty = PrettyPresence{
			DiscordUser:   discordUser,
			DiscordStatus: "offline",
			Activities:    []any{},
			KV:            kv,
		}
	}

	cacheKey := userID
	if discordUser != nil {
		if id, ok := discordUser["id"].(string); ok {
			cacheKey = id
		}
	}
	cacheSet(cacheKey, pretty)
	return pretty
}

// GetPresence returns the raw presence or a typed not-monitored error.
func GetPresence(userID string) (*Presence, *Error) {
	if p, ok := Reg.Lookup(userID); ok {
		return p, nil
	}
	return nil, &Error{HTTPCode: 404, Code: "user_not_monitored", Message: "User is not being monitored by Lanyard"}
}

// GetPrettyPresence returns the cached pretty presence, building it from raw
// state on a cache miss. Mirrors Lanyard.Presence.get_pretty_presence.
func GetPrettyPresence(userID string) (*PrettyPresence, *Error) {
	if c, ok := cacheGet(userID); ok {
		return &c, nil
	}
	p, err := GetPresence(userID)
	if err != nil {
		return nil, err
	}
	pretty := p.BuildPretty()
	return &pretty, nil
}

// GetKV returns the KV map for a user (errors if not monitored).
func GetKV(userID string) (map[string]string, *Error) {
	p, err := GetPresence(userID)
	if err != nil {
		return nil, err
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.KV == nil {
		return map[string]string{}, nil
	}
	// return a copy to avoid races on the caller side
	out := make(map[string]string, len(p.KV))
	for k, v := range p.KV {
		out[k] = v
	}
	return out, nil
}

// SubscribeToIDsAndBuild subscribes sub to each monitored id and returns a
// user_id -> pretty presence map. Mirrors subscribe_to_ids_and_build.
func SubscribeToIDsAndBuild(ids []string, sub Subscriber) map[string]any {
	out := make(map[string]any)
	for _, id := range ids {
		if p, ok := Reg.Lookup(id); ok {
			pretty := p.BuildPretty()
			p.AddSubscriber(sub)
			out[id] = pretty
		}
	}
	return out
}

// SubscribeToID subscribes sub to a single id, returning its pretty presence
// (or nil if not monitored).
func SubscribeToID(id string, sub Subscriber) *PrettyPresence {
	p, ok := Reg.Lookup(id)
	if !ok {
		return nil
	}
	pretty := p.BuildPretty()
	p.AddSubscriber(sub)
	return &pretty
}

// Unsubscribe removes sub from a presence's subscriber set.
func Unsubscribe(id string, sub Subscriber) {
	if p, ok := Reg.Lookup(id); ok {
		p.RemoveSubscriber(sub)
	}
}

// --- subscriber management ---

// AddSubscriber registers a subscriber for this presence.
func (p *Presence) AddSubscriber(sub Subscriber) {
	p.mu.Lock()
	p.subscribers[sub] = struct{}{}
	p.mu.Unlock()
}

// RemoveSubscriber unregisters a subscriber from this presence.
func (p *Presence) RemoveSubscriber(sub Subscriber) {
	p.mu.Lock()
	delete(p.subscribers, sub)
	p.mu.Unlock()
}

func (p *Presence) subscriberList() []Subscriber {
	subs := make([]Subscriber, 0, len(p.subscribers))
	for s := range p.subscribers {
		subs = append(subs, s)
	}
	return subs
}

// --- global subscribers ---

// AddGlobalSubscriber registers a subscribe_to_all socket.
func AddGlobalSubscriber(s Subscriber) {
	globalMu.Lock()
	globalSubs[s] = struct{}{}
	globalMu.Unlock()
}

// RemoveGlobalSubscriber removes a socket from the global subscriber set.
func RemoveGlobalSubscriber(s Subscriber) {
	globalMu.Lock()
	delete(globalSubs, s)
	globalMu.Unlock()
}

func globalSubscribers() []Subscriber {
	globalMu.RLock()
	defer globalMu.RUnlock()
	subs := make([]Subscriber, 0, len(globalSubs))
	for s := range globalSubs {
		subs = append(subs, s)
	}
	return subs
}

// --- cache helpers ---

func cacheGet(id string) (PrettyPresence, bool) {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	c, ok := cache[id]
	return c, ok
}

func cacheSet(id string, p PrettyPresence) {
	cacheMu.Lock()
	cache[id] = p
	cacheMu.Unlock()
}

func cacheDelete(id string) {
	cacheMu.Lock()
	delete(cache, id)
	cacheMu.Unlock()
}

// --- conversion helpers ---

func clientStatusHas(dpres map[string]any, key string) bool {
	cs, ok := dpres["client_status"].(map[string]any)
	if !ok {
		return false
	}
	_, has := cs[key]
	return has
}

func asStringMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func asKV(v any) map[string]string {
	switch m := v.(type) {
	case map[string]string:
		return m
	case map[string]any:
		out := make(map[string]string, len(m))
		for k, val := range m {
			out[k] = fmt.Sprint(val)
		}
		return out
	}
	return map[string]string{}
}
