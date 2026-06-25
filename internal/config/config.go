// Package config loads runtime configuration from environment variables,
// mirroring the Elixir project's config/runtime.exs and config/dev.exs.
package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the zRuvix service.
type Config struct {
	HTTPPort               int
	DiscordSpotifyActivity string
	CommandPrefix          string
	BotPresence            string
	BotPresenceType        int
	BotToken               string
	RedisURI               string
	IsIdempotent           bool
	ExternalURL            string
}

// C is the process-wide configuration, populated by Load at startup. It is
// the Go equivalent of Elixir's Application.get_env(:zRuvix, ...).
var C *Config

// Load reads configuration from the environment and stores it in C. If a .env
// file is present in the working directory it is loaded first; real environment
// variables always take precedence over .env values (godotenv does not
// override already-set variables), so systemd/shell exports still win on the
// VPS.
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		// Not fatal: in production you may rely solely on real env vars.
		log.Printf("config: no .env file loaded (%v); using environment variables", err)
	}

	c := &Config{
		HTTPPort:               envInt("PORT", 4001),
		DiscordSpotifyActivity: "spotify:1",
		CommandPrefix:          envStr("COMMAND_PREFIX", "?"),
		BotPresence:            envStr("BOT_PRESENCE", "you <3"),
		BotPresenceType:        envInt("BOT_PRESENCE_TYPE", 3),
		BotToken:               os.Getenv("BOT_TOKEN"),
		RedisURI:               redisURI(),
		IsIdempotent:           os.Getenv("IS_IDEMPOTENT") == "true",
		ExternalURL:            envStr("EXTERNAL_URL", "http://127.0.0.1:4001"),
	}
	C = c
	return c
}

// redisURI mirrors the Elixir fallback chain: REDIS_DSN || REDIS_URI ||
// REDIS_URL, finally falling back to redis://REDIS_HOST:6379.
func redisURI() string {
	for _, k := range []string{"REDIS_DSN", "REDIS_URI", "REDIS_URL"} {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("redis://%s:6379", host)
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
