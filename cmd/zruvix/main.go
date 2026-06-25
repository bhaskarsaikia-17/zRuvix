// Command zruvix is zRuvix: a Go service exposing Discord presences as a REST
// API + WebSocket gateway backed by Redis. It is a port of the Lanyard Elixir
// project and mirrors the supervision tree set up in Lanyard.start/2.
package main

import (
	"fmt"
	"log"
	"net/http"

	"zruvix/internal/api"
	"zruvix/internal/bot"
	"zruvix/internal/config"
	"zruvix/internal/gateway"
	"zruvix/internal/presence"
	"zruvix/internal/redis"
)

func main() {
	cfg := config.Load()

	// Connect Redis and wire the cross-node global_sync handler to the
	// presence registry (kept as a callback to avoid an import cycle).
	if err := redis.Connect(cfg.RedisURI); err != nil {
		log.Fatalf("failed to configure redis: %v", err)
	}
	redis.GlobalSyncHandler = func(userID string, diff map[string]any) {
		presence.Reg.Sync(userID, diff, true)
	}

	// Route Discord MESSAGE_CREATE events to the bot command handler.
	gateway.OnMessageCreate = bot.HandleMessage

	// Start the Discord gateway client (the bot) in the background.
	if cfg.BotToken != "" {
		go gateway.Run(cfg.BotToken)
	} else {
		log.Println("BOT_TOKEN not set; Discord gateway client will not start")
	}

	addr := fmt.Sprintf(":%d", cfg.HTTPPort)
	log.Printf("zRuvix listening on %s", addr)
	if err := http.ListenAndServe(addr, api.Router()); err != nil {
		log.Fatalf("http server error: %v", err)
	}
}
