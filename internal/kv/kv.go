// Package kv implements the Lanyard KV store interface (Lanyard.KV.Interface),
// layering validation and presence syncing over the Redis hash store.
package kv

import (
	"errors"
	"regexp"

	"zruvix/internal/presence"
	"zruvix/internal/redis"
)

var keyPattern = regexp.MustCompile(`^[a-zA-Z0-9_]*$`)

// GetAll returns every key/value pair for a user.
func GetAll(userID string) (map[string]string, error) {
	kv, perr := presence.GetKV(userID)
	if perr != nil {
		return nil, perr
	}
	return kv, nil
}

// Get returns a single value, erroring if the key is absent.
func Get(userID, key string) (string, error) {
	kv, err := GetAll(userID)
	if err != nil {
		return "", err
	}
	if v, ok := kv[key]; ok {
		return v, nil
	}
	return "", errors.New("Key " + key + " not found in KV")
}

// Set validates and stores a single pair, then syncs the presence. It enforces
// the 512-key limit, returning the stored value on success.
func Set(userID, key, value string) (string, error) {
	kv, err := GetAll(userID)
	if err != nil {
		return "", err
	}
	if len(kv) > 511 {
		return "", errors.New("request would exceed key limit (512), please delete keys first")
	}
	if verr := ValidatePair(key, value); verr != nil {
		return "", verr
	}

	redis.HSet("lanyard_kv:"+userID, key, value)
	kv[key] = value
	presence.Reg.Sync(userID, map[string]any{"kv": kv}, false)
	return value, nil
}

// Multiset merges a map of pairs and syncs. Callers are expected to validate
// pairs beforehand (the router does this), matching the Elixir flow.
func Multiset(userID string, m map[string]string) error {
	redis.HSetMap("lanyard_kv:"+userID, m)

	full, err := GetAll(userID)
	if err != nil {
		return err
	}
	for k, v := range m {
		full[k] = v
	}
	presence.Reg.Sync(userID, map[string]any{"kv": full}, false)
	return nil
}

// Del removes a key and syncs the presence.
func Del(userID, key string) error {
	redis.HDel("lanyard_kv:"+userID, key)

	kv, err := GetAll(userID)
	if err != nil {
		return err
	}
	delete(kv, key)
	presence.Reg.Sync(userID, map[string]any{"kv": kv}, false)
	return nil
}

// ValidatePair enforces the KV key/value constraints from the README.
func ValidatePair(key, value string) error {
	switch {
	case len(key) > 255:
		return errors.New("key must be 255 characters or less")
	case !keyPattern.MatchString(key):
		return errors.New("key must be alphanumeric (a-zA-Z0-9_)")
	case len(value) > 30000:
		return errors.New("value must be 30000 characters or less")
	default:
		return nil
	}
}
