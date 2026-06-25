package bot

import (
	"fmt"
	"strings"
	"time"

	"zruvix/internal/config"
	"zruvix/internal/kv"
	"zruvix/internal/presence"
	"zruvix/internal/version"
)

// startTime is captured at package init for uptime reporting.
var startTime = time.Now()

// commandFunc handles a parsed command's args plus the raw message data.
type commandFunc func(args []string, data map[string]any)

func commandMap() map[string]commandFunc {
	return map[string]commandFunc{
		// KV store
		"set":   handleSet,
		"get":   handleGet,
		"del":   handleDel,
		"kv":    handleKV,
		"list":  handleKV,
		"count": handleCount,
		"clear": handleClear,
		// Presence / service
		"me":       handleMe,
		"presence": handleMe,
		"stats":    handleStats,
		"ping":     handlePing,
		// API / meta
		"apikey": handleAPIKey,
		"invite": handleInvite,
		"help":   handleHelp,
	}
}

// HandleMessage is the entry point for MESSAGE_CREATE events. It ignores bots,
// checks the command prefix, parses the command and dispatches it. Mirrors
// zRuvix.DiscordBot.CommandHandler.handle_message.
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
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	if fn, ok := commandMap()[cmd]; ok {
		fn(args, data)
	}
}

// --- KV commands -------------------------------------------------------------

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
			SendErrorMsg(channelID, err.Error())
			return
		}
		SendSuccess(channelID, "✅ Saved",
			fmt.Sprintf("Saved **%s**.\nRead it with `%sget %s`, or fetch all your data at %s/v1/users/%s",
				key, config.C.CommandPrefix, key, config.C.ExternalURL, userID))
		return
	}

	if len(args) == 1 {
		if validateAPIKey(userID, args[0]) {
			warnLeakedKey(channelID, userID)
			return
		}
		SendEmbed(channelID, Embed("How to save info",
			fmt.Sprintf("`%[1]sset <label> <value>`\nExample: `%[1]sset location Tokyo`", config.C.CommandPrefix), ColorBrand))
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
		SendErrorMsg(channelID, err.Error())
		return
	}
	safe := strings.ReplaceAll(v, "`", "`\u200b")
	SendEmbed(channelID, Embed("🔑 "+key, "```\n"+safe+"\n```", ColorSuccess))
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
	SendSuccess(channelID, "🗑️ Removed", fmt.Sprintf("Removed **%s**.", key))
}

func handleKV(_ []string, data map[string]any) {
	userID := authorID(data)
	channelID := stringField(data, "channel_id")

	all, err := kv.GetAll(userID)
	names := "*You haven't saved anything yet.*"
	if err == nil && len(all) > 0 {
		list := make([]string, 0, len(all))
		for k := range all {
			list = append(list, k)
		}
		names = "```\n" + strings.Join(list, ", ") + "\n```"
	}

	p := config.C.CommandPrefix
	embed := Embed("📦 Your saved info", "These are the labels you've saved. Each holds a value you can read back or share via the API.", ColorBrand)
	embed["fields"] = []any{
		field("Labels", names, false),
		field("Manage", fmt.Sprintf("`%[1]sset <label> <value>` — save\n`%[1]sget <label>` — read\n`%[1]sdel <label>` — remove\n`%[1]sclear confirm` — remove all", p), false),
	}
	SendEmbed(channelID, embed)
}

func handleCount(_ []string, data map[string]any) {
	userID := authorID(data)
	channelID := stringField(data, "channel_id")

	all, err := kv.GetAll(userID)
	if err != nil {
		SendErrorMsg(channelID, err.Error())
		return
	}
	SendEmbed(channelID, Embed("🔢 Saved items",
		fmt.Sprintf("You've saved **%d** item(s) (max 512).", len(all)), ColorBrand))
}

func handleClear(args []string, data map[string]any) {
	userID := authorID(data)
	channelID := stringField(data, "channel_id")

	if len(args) == 0 || strings.ToLower(args[0]) != "confirm" {
		SendEmbed(channelID, Embed("⚠️ Confirm deletion",
			fmt.Sprintf("This deletes **everything** you've saved and cannot be undone.\nRun `%sclear confirm` to proceed.", config.C.CommandPrefix),
			ColorError))
		return
	}
	if err := kv.Clear(userID); err != nil {
		SendErrorMsg(channelID, err.Error())
		return
	}
	SendSuccess(channelID, "🧹 Cleared", "Everything you saved has been deleted.")
}

// --- presence / service commands --------------------------------------------

func handleMe(_ []string, data map[string]any) {
	userID := authorID(data)
	channelID := stringField(data, "channel_id")

	p, perr := presence.GetPrettyPresence(userID)
	if perr != nil {
		SendErrorMsg(channelID, perr.Message+" (share a server with me so I can track you)")
		return
	}

	username := "your"
	if p.DiscordUser != nil {
		if u, ok := p.DiscordUser["username"].(string); ok {
			username = u
		}
	}

	embed := Embed("👤 Presence — "+username, "", ColorBrand)
	fields := []any{
		field("Status", statusEmoji(p.DiscordStatus)+" "+p.DiscordStatus, true),
		field("Platforms", platformList(p), true),
		field("Saved items", fmt.Sprintf("%d", len(p.KV)), true),
	}
	if p.ListeningToSpotify && p.Spotify != nil {
		fields = append(fields, field("🎵 Spotify",
			fmt.Sprintf("**%v**\nby %v", p.Spotify.Song, p.Spotify.Artist), false))
	}
	if p.ListeningToYouTubeMusic && p.YouTubeMusic != nil {
		fields = append(fields, field("📺 YouTube Music",
			fmt.Sprintf("**%v**\nby %v", p.YouTubeMusic.Song, p.YouTubeMusic.Artist), false))
	}
	fields = append(fields, field("Activities", fmt.Sprintf("%d active", len(p.Activities)), true))
	embed["fields"] = fields

	if p.DiscordUser != nil {
		if av, ok := p.DiscordUser["avatar"].(string); ok && av != "" {
			embed["thumbnail"] = map[string]any{
				"url": fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png?size=256", userID, av),
			}
		}
	}
	SendEmbed(channelID, embed)
}

func handleStats(_ []string, data map[string]any) {
	channelID := stringField(data, "channel_id")
	up := time.Since(startTime).Round(time.Second)

	embed := Embed("📊 zRuvix Stats", "", ColorBrand)
	embed["fields"] = []any{
		field("Monitored users", fmt.Sprintf("%d", presence.Reg.Count()), true),
		field("Uptime", up.String(), true),
		field("Version", version.Version, true),
	}
	SendEmbed(channelID, embed)
}

func handlePing(_ []string, data map[string]any) {
	channelID := stringField(data, "channel_id")
	SendEmbed(channelID, Embed("🏓 Pong!", "zRuvix is online and handling commands.", ColorSuccess))
}

// --- meta commands -----------------------------------------------------------

func handleInvite(_ []string, data map[string]any) {
	channelID := stringField(data, "channel_id")
	base := config.C.ExternalURL
	SendEmbed(channelID, Embed("🔗 zRuvix",
		fmt.Sprintf("**Your data (API):** %s/v1/users/<your-id>\n**Realtime (WebSocket):** %s/socket", base, base),
		ColorBrand))
}

func handleHelp(_ []string, data map[string]any) {
	channelID := stringField(data, "channel_id")
	p := config.C.CommandPrefix

	embed := Embed("zRuvix — Commands", "Attach custom info to your Discord profile (like your location, mood, or website) and read it back from a public API.", ColorBrand)
	embed["fields"] = []any{
		field("📝 Save your info", fmt.Sprintf(
			"`%[1]sset <label> <value>` — save something\n"+
				"`%[1]sget <label>` — read it back\n"+
				"`%[1]sdel <label>` — remove one\n"+
				"`%[1]slist` — show everything you saved\n"+
				"`%[1]scount` — how many you've saved\n"+
				"`%[1]sclear confirm` — remove everything", p), false),
		field("👤 Presence", fmt.Sprintf(
			"`%[1]sme` — show your live presence\n"+
				"`%[1]sstats` — service stats\n"+
				"`%[1]sping` — is the bot alive?", p), false),
		field("🔑 API access", fmt.Sprintf(
			"`%[1]sapikey` — get your secret key (DM only) to edit data over HTTP\n"+
				"`%[1]sinvite` — your API links\n"+
				"`%[1]shelp` — this message", p), false),
	}
	SendEmbed(channelID, embed)
}

// warnLeakedKey tells the user they posted their key and rotates it.
func warnLeakedKey(channelID, userID string) {
	SendErrorMsg(channelID, "Whoops, you just posted your API key. It's meant to stay private — regenerating it now, check your DMs.")
	regenerateAndDM(userID)
}

// --- helpers -----------------------------------------------------------------

func statusEmoji(status string) string {
	switch status {
	case "online":
		return "🟢"
	case "idle":
		return "🟡"
	case "dnd":
		return "🔴"
	default:
		return "⚪"
	}
}

func platformList(p *presence.PrettyPresence) string {
	var on []string
	if p.ActiveOnDiscordDesktop {
		on = append(on, "Desktop")
	}
	if p.ActiveOnDiscordMobile {
		on = append(on, "Mobile")
	}
	if p.ActiveOnDiscordWeb {
		on = append(on, "Web")
	}
	if len(on) == 0 {
		return "—"
	}
	return strings.Join(on, ", ")
}

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
