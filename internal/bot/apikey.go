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

	SendEmbed(channelID, map[string]any{
		"title": "zRuvix API Key",
		"description": "**Absolutely do not share or post this key anywhere, it is a secret key that will allow anyone to manage your zRuvix K/V**\n\n" +
			"**This key is not to be used in a front-end application/website**\n\n" +
			"If you are looking for the public endpoint for your data, you would use your discord user ID like so\n" +
			config.C.ExternalURL + "/v1/users/" + userID,
		"color":  0x5865F2,
		"footer": map[string]any{"text": "Run this command again if you need to re-generate your key"},
		"fields": []any{
			map[string]any{"name": "Key", "value": "||`" + key + "`||\n-# *click above to reveal*", "inline": false},
		},
	})
}

// regenerateAndDM rotates the key and DMs the user, used when a user leaks their
// key inside a KV command.
func regenerateAndDM(userID string) {
	key := storeNewKey(userID)
	dmChannel := CreateDM(userID)
	if dmChannel == "" {
		return
	}
	SendMessage(dmChannel,
		":repeat: **We've regenerated your api key as you used it in a K/V command.**\n"+
			"Your new zRuvix API key is `"+key+"`\n\n"+
			"**ABSOLUTELY DO NOT SHARE OR POST THIS KEY ANYWHERE IT WILL ALLOW ANYONE TO MANAGE YOUR zRuvix K/V**\n"+
			"*Run `.apikey` in this DM if you need to re-generate your key*")
}
