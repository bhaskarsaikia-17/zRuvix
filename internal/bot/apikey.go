package bot

import (
	"math/rand"

	"zruvix/internal/config"
	"zruvix/internal/redis"
)

const hexSymbols = "0123456789abcdef"

// generateAPIKey returns a random 32-character hex key.
func generateAPIKey() string {
	b := make([]byte, 32)
	for i := range b {
		b[i] = hexSymbols[rand.Intn(len(hexSymbols))]
	}
	return string(b)
}

// validateAPIKey reports whether key is the user's current API key.
func validateAPIKey(userID, key string) bool {
	return redis.Get("user_api_key:"+userID) == key
}

// storeNewKey rotates a user's API key in Redis, deleting any previous mapping.
func storeNewKey(userID string) string {
	key := generateAPIKey()
	if existing := redis.Get("user_api_key:" + userID); existing != "" {
		redis.Del("api_key:" + existing)
	}
	redis.Set("api_key:"+key, userID)
	redis.Set("user_api_key:"+userID, key)
	return key
}

// handleAPIKey implements the .apikey command. It only works in DMs.
func handleAPIKey(_ []string, data map[string]any) {
	channelID := stringField(data, "channel_id")
	// In a guild, refuse (guild_id present).
	if _, inGuild := data["guild_id"]; inGuild && data["guild_id"] != nil {
		SendMessage(channelID, ":x: You can only perform this command in DMs with me")
		return
	}

	userID := authorID(data)
	key := storeNewKey(userID)

	embed := Embed("🔑 Your zRuvix API key",
		"This key lets you edit your saved data over HTTP.\n\n"+
			"**Keep it secret** — anyone with it can change your data. Don't put it in a website or front-end.\n\n"+
			"Your public data lives at:\n"+config.C.ExternalURL+"/v1/users/"+userID,
		ColorBrand)
	embed["footer"] = map[string]any{"text": "Run this command again to generate a new key"}
	embed["fields"] = []any{
		field("Key", "||`"+key+"`||\n-# *click above to reveal*", false),
	}
	SendEmbed(channelID, embed)
}

// regenerateAndDM rotates the key and DMs the user, used when a user leaks their
// key inside a command.
func regenerateAndDM(userID string) {
	key := storeNewKey(userID)
	dmChannel := CreateDM(userID)
	if dmChannel == "" {
		return
	}
	SendMessage(dmChannel,
		":repeat: **We regenerated your API key because you posted the old one in a command.**\n"+
			"Your new zRuvix API key is `"+key+"`\n\n"+
			"**DO NOT SHARE OR POST THIS KEY ANYWHERE — anyone with it can change your saved data.**\n"+
			"*Run `"+config.C.CommandPrefix+"apikey` in this DM to generate a new one.*")
}
