// Package redis wraps a go-redis client and reproduces the command surface of
// zRuvix.Connectivity.Redis, including the zruvix:global_sync pub/sub bridge.
package redis

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"

	goredis "github.com/redis/go-redis/v9"
)

// NodeID identifies this process in cross-node global_sync messages. It mirrors
// :erlang.phash2(node()) — a stable identifier unique to this instance.
var NodeID = rand.Int()

// GlobalSyncHandler is invoked when a global_sync message arrives from another
// node. It is wired up at startup (by main) to presence.Sync, which keeps the
// redis package free of a dependency on the presence package.
var GlobalSyncHandler func(userID string, diff map[string]any)

var (
	client *goredis.Client
	ctx    = context.Background()
)

// Connect dials Redis using the provided URI and starts the global_sync
// subscriber. It returns an error if the URI cannot be parsed.
func Connect(uri string) error {
	opts, err := goredis.ParseURL(uri)
	if err != nil {
		return err
	}
	client = goredis.NewClient(opts)
	go subscribeGlobalSync()
	return nil
}

func subscribeGlobalSync() {
	sub := client.Subscribe(ctx, "zruvix:global_sync")
	log.Println("Redis: subscribed to zruvix:global_sync")
	for msg := range sub.Channel() {
		var payload struct {
			NodeID int            `json:"node_id"`
			UserID string         `json:"user_id"`
			Diff   map[string]any `json:"diff"`
		}
		if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
			log.Printf("Redis: unknown payload format: %s", msg.Payload)
			continue
		}
		// Ignore messages emitted by this same node.
		if payload.NodeID == NodeID {
			continue
		}
		if GlobalSyncHandler != nil {
			GlobalSyncHandler(payload.UserID, payload.Diff)
		}
	}
}

// Get returns the string value at key, or "" if the key is missing.
func Get(key string) string {
	v, err := client.Get(ctx, key).Result()
	if err != nil {
		return ""
	}
	return v
}

// Set stores value at key.
func Set(key, value string) {
	client.Set(ctx, key, value, 0)
}

// Del removes key.
func Del(key string) {
	client.Del(ctx, key)
}

// HGetAll returns the full hash at key as a map (empty if missing).
func HGetAll(key string) map[string]string {
	v, err := client.HGetAll(ctx, key).Result()
	if err != nil || v == nil {
		return map[string]string{}
	}
	return v
}

// HGet returns a single hash field value, or "" if missing.
func HGet(key, field string) string {
	v, err := client.HGet(ctx, key, field).Result()
	if err != nil {
		return ""
	}
	return v
}

// HSet sets one field/value pair in the hash at key.
func HSet(key, field, value string) {
	client.HSet(ctx, key, field, value)
}

// HSetMap merges the given map into the hash at key.
func HSetMap(key string, values map[string]string) {
	if len(values) == 0 {
		return
	}
	args := make([]any, 0, len(values)*2)
	for k, v := range values {
		args = append(args, k, v)
	}
	client.HSet(ctx, key, args...)
}

// HDel removes a field from the hash at key.
func HDel(key, field string) {
	client.HDel(ctx, key, field)
}

// HIncrBy atomically increments a hash field by amount.
func HIncrBy(key, field string, amount int64) {
	client.HIncrBy(ctx, key, field, amount)
}

// Publish sends a message to a pub/sub channel.
func Publish(channel, message string) {
	client.Publish(ctx, channel, message)
}
