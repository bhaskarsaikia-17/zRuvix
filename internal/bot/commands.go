package bot

import (
	"strings"

	"zruvix/internal/config"
	"zruvix/internal/kv"
)

// commandFunc handles a parsed command's args plus the raw message data.
type commandFunc func(args []string, data map[string]any)

func commandMap() map[string]commandFunc {
	return map[string]commandFunc{
		"get":    handleGet,
		"set":    handleSet,
		"del":    handleDel,
		"apikey": handleAPIKey,
		"kv":     handleKV,
		"help":   handleKV,
	}
}

// HandleMessage is the entry point for MESSAGE_CREATE events. It ignores bots,
// checks the command prefix, parses the command and dispatches it. Mirrors
// Lanyard.DiscordBot.CommandHandler.handle_message.
func HandleMessage(data map[string]any) {
	// Ignore messages from other bots.
	if author, ok := data["author"].(map[string]any); ok {
		if isBot, _ := author["bot"].(bool); isBot {
			return
		}
	}

	content, ok := data["content"].(string)
	if !ok {
		return
	}

	prefix := config.C.CommandPrefix
	if !strings.HasPrefix(content, prefix) {
		return
	}

	rest := strings.TrimPrefix(content, prefix)
	parts := strings.Split(rest, " ")
	cmd := parts[0]
	args := parts[1:]

	if fn, ok := commandMap()[cmd]; ok {
		fn(args, data)
	}
}

// --- commands ---

func handleSet(args []string, data map[string]any) {
	userID := authorID(data)
	channelID := stringField(data, "channel_id")

	if len(args) >= 2 {
		key := args[0]
		value := strings.Join(args[1:], " ")

		if validateAPIKey(userID, key) {
			warnLeakedKey(channelID, userID)
			return
		}
		if _, err := kv.Set(userID, key, value); err != nil {
			SendMessage(channelID, ":x: "+err.Error())
			return
		}
		SendMessage(channelID, ":white_check_mark: `"+key+"` was set. View it with `"+
			config.C.CommandPrefix+"get "+key+"` or go to "+config.C.ExternalURL+"/v1/users/"+userID)
		return
	}

	// Invalid usage (one arg): warn if it's the user's key, else show usage.
	if len(args) == 1 {
		if validateAPIKey(userID, args[0]) {
			warnLeakedKey(channelID, userID)
			return
		}
		SendMessage(channelID, "Invalid usage. Example `set` command usage:\n`"+
			config.C.CommandPrefix+"set <key> <value>`")
	}
}

func handleGet(args []string, data map[string]any) {
	if len(args) != 1 {
		return
	}
	userID := authorID(data)
	channelID := stringField(data, "channel_id")
	key := args[0]

	if validateAPIKey(userID, key) {
		warnLeakedKey(channelID, userID)
		return
	}
	v, err := kv.Get(userID, key)
	if err != nil {
		SendMessage(channelID, ":x: "+err.Error())
		return
	}
	safe := strings.ReplaceAll(v, "`", "`\u200b")
	SendMessage(channelID, ":white_check_mark: Key: `"+key+"` | Value: ```"+safe+"```")
}

func handleDel(args []string, data map[string]any) {
	if len(args) != 1 {
		return
	}
	userID := authorID(data)
	channelID := stringField(data, "channel_id")
	key := args[0]

	if validateAPIKey(userID, key) {
		warnLeakedKey(channelID, userID)
		return
	}
	_ = kv.Del(userID, key)
	SendMessage(channelID, ":white_check_mark: Deleted key: `"+key+"`")
}

func handleKV(_ []string, data map[string]any) {
	userID := authorID(data)
	channelID := stringField(data, "channel_id")

	all, err := kv.GetAll(userID)
	keys := "No keys"
	if err == nil && len(all) > 0 {
		names := make([]string, 0, len(all))
		for k := range all {
			names = append(names, k)
		}
		keys = strings.Join(names, ", ")
	}

	p := config.C.CommandPrefix
	SendMessage(channelID,
		"*`"+p+"get <key>` to get a value*\n"+
			"*`"+p+"del <key>` to delete an existing key*\n"+
			"*`"+p+"set <key> <value>` to set a key*\n\n"+
			"**Keys:** ```"+keys+"```")
}

// warnLeakedKey tells the user they posted their key and rotates it.
func warnLeakedKey(channelID, userID string) {
	SendMessage(channelID, ":x: Whoops, you just posted your API key, this is meant to stay private, regenerating this for you, check your DM")
	regenerateAndDM(userID)
}

// --- helpers ---

func authorID(data map[string]any) string {
	if author, ok := data["author"].(map[string]any); ok {
		if id, ok := author["id"].(string); ok {
			return id
		}
	}
	return ""
}

func stringField(data map[string]any, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}
