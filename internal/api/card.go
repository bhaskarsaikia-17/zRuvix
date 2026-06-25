package api

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"zruvix/internal/presence"
)

// statusColors maps Discord statuses to their dot colours.
var statusColors = map[string]string{
	"online":  "#23a55a",
	"idle":    "#f0b232",
	"dnd":     "#f23f43",
	"offline": "#80848e",
}

// handleCard renders a live, self-contained SVG status card suitable for
// embedding in a GitHub README or any website via <img src=".../card.svg">.
func handleCard(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	name := "Unknown"
	status := "offline"
	secondary := "Not being tracked"
	avatar := "https://cdn.discordapp.com/embed/avatars/0.png"
	var artURL string

	if p, err := presence.GetPrettyPresence(id); err == nil {
		status = p.DiscordStatus
		if p.DiscordUser != nil {
			if u, ok := p.DiscordUser["username"].(string); ok && u != "" {
				name = u
			}
			if av, ok := p.DiscordUser["avatar"].(string); ok && av != "" {
				avatar = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png?size=128", id, av)
			}
		}
		secondary = cardSecondary(p)
		if p.NowPlaying != nil && p.NowPlaying.AlbumArtURL != nil {
			artURL = *p.NowPlaying.AlbumArtURL
		}
	}

	color := statusColors[status]
	if color == "" {
		color = statusColors["offline"]
	}

	svg := renderCardSVG(cardData{
		Name:      truncate(name, 20),
		Secondary: truncate(secondary, 34),
		Status:    status,
		Color:     color,
		Avatar:    avatar,
		ArtURL:    artURL,
	})

	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	// Keep it live: don't let proxies cache a stale presence.
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(svg))
}

type cardData struct {
	Name      string
	Secondary string
	Status    string
	Color     string
	Avatar    string
	ArtURL    string
}

func renderCardSVG(d cardData) string {
	art := ""
	if d.ArtURL != "" {
		art = fmt.Sprintf(
			`<clipPath id="art"><rect x="386" y="30" width="80" height="80" rx="12"/></clipPath>`+
				`<image x="386" y="30" width="80" height="80" href="%s" clip-path="url(#art)" preserveAspectRatio="xMidYMid slice"/>`,
			html.EscapeString(d.ArtURL))
	}

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="480" height="140" viewBox="0 0 480 140" role="img">
  <rect width="480" height="140" rx="16" fill="#1e1f22"/>
  <rect x="0.5" y="0.5" width="479" height="139" rx="16" fill="none" stroke="#2b2d31"/>
  <clipPath id="av"><circle cx="62" cy="62" r="40"/></clipPath>
  <image x="22" y="22" width="80" height="80" href="%s" clip-path="url(#av)" preserveAspectRatio="xMidYMid slice"/>
  <circle cx="62" cy="62" r="40" fill="none" stroke="#2b2d31" stroke-width="2"/>
  <circle cx="90" cy="90" r="13" fill="#1e1f22"/>
  <circle cx="90" cy="90" r="9" fill="%s">
    <animate attributeName="opacity" values="1;0.5;1" dur="2s" repeatCount="indefinite"/>
  </circle>
  <text x="130" y="58" font-family="Segoe UI,Helvetica,Arial,sans-serif" font-size="22" font-weight="700" fill="#ffffff">%s</text>
  <text x="130" y="86" font-family="Segoe UI,Helvetica,Arial,sans-serif" font-size="15" fill="#b5bac1">%s</text>
  <text x="130" y="112" font-family="Segoe UI,Helvetica,Arial,sans-serif" font-size="11" fill="#6d6f78">zRuvix • %s</text>
  %s
</svg>`,
		html.EscapeString(d.Avatar),
		d.Color,
		html.EscapeString(d.Name),
		html.EscapeString(d.Secondary),
		html.EscapeString(d.Status),
		art)
}

// cardSecondary chooses the most interesting line: now playing > activity > status.
func cardSecondary(p *presence.PrettyPresence) string {
	if p.NowPlaying != nil {
		song := anyToString(p.NowPlaying.Song)
		artist := anyToString(p.NowPlaying.Artist)
		switch {
		case song != "" && artist != "":
			return "♪ " + song + " — " + artist
		case song != "":
			return "♪ " + song
		}
	}
	for _, a := range p.Activities {
		if m, ok := a.(map[string]any); ok {
			if n, ok := m["name"].(string); ok && n != "" {
				return n
			}
		}
	}
	switch p.DiscordStatus {
	case "online":
		return "Online"
	case "idle":
		return "Idle"
	case "dnd":
		return "Do Not Disturb"
	default:
		return "Offline"
	}
}

func anyToString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}
