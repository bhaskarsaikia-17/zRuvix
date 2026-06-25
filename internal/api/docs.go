package api

import (
	"html"
	"net/http"
	"strings"

	"zruvix/internal/config"
	"zruvix/internal/version"
)

// handleDocs serves a self-contained HTML documentation page at /v1/docs.
// Base URL, command prefix, version and changelog are injected from config.
func handleDocs(w http.ResponseWriter, _ *http.Request) {
	page := docsHTML
	page = strings.ReplaceAll(page, "{{BASE}}", config.C.ExternalURL)
	page = strings.ReplaceAll(page, "{{PREFIX}}", config.C.CommandPrefix)
	page = strings.ReplaceAll(page, "{{VERSION}}", html.EscapeString(version.Version))
	page = strings.ReplaceAll(page, "{{CHANGELOG}}", renderChangelog())

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(page))
}

// handleVersion returns the current version and full changelog as JSON.
func handleVersion(w http.ResponseWriter, _ *http.Request) {
	respondOK(w, map[string]any{
		"version":   version.Version,
		"changelog": version.Changelog,
	})
}

// renderChangelog builds the changelog HTML from version.Changelog.
func renderChangelog() string {
	var b strings.Builder
	for _, rel := range version.Changelog {
		b.WriteString(`<div class="card"><h3 id="v-` + html.EscapeString(rel.Version) + `">v` +
			html.EscapeString(rel.Version) + ` <span class="pill">` + html.EscapeString(rel.Date) + `</span></h3>`)
		if rel.Title != "" {
			b.WriteString(`<p class="lead" style="font-size:15px;margin:0 0 8px">` + html.EscapeString(rel.Title) + `</p>`)
		}
		b.WriteString("<ul>")
		for _, c := range rel.Changes {
			b.WriteString("<li>" + html.EscapeString(c) + "</li>")
		}
		b.WriteString("</ul></div>")
	}
	return b.String()
}

const docsHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>zRuvix — API Documentation</title>
<style>
:root{--bg:#0b0c0e;--panel:#151619;--panel2:#1b1d21;--border:#26282d;--text:#e6e7ea;--muted:#9aa0a8;--accent:#7c5cff;--green:#23a55a;--code:#0f1013}
*{box-sizing:border-box}
body{margin:0;font-family:Inter,Segoe UI,Helvetica,Arial,sans-serif;background:var(--bg);color:var(--text);line-height:1.6}
a{color:var(--accent);text-decoration:none}
a:hover{text-decoration:underline}
.layout{display:flex;min-height:100vh}
aside{width:260px;flex:none;border-right:1px solid var(--border);padding:24px 18px;position:sticky;top:0;height:100vh;overflow-y:auto;background:var(--panel)}
aside .brand{font-size:20px;font-weight:800;letter-spacing:-.3px;margin-bottom:4px}
aside .tag{color:var(--muted);font-size:12px;margin-bottom:20px}
aside nav a{display:block;color:var(--muted);padding:6px 10px;border-radius:8px;font-size:14px}
aside nav a:hover{background:var(--panel2);color:var(--text);text-decoration:none}
aside nav .group{margin:14px 0 4px;color:#6b7079;font-size:11px;text-transform:uppercase;letter-spacing:.08em}
main{flex:1;max-width:860px;padding:48px 40px 120px;margin:0 auto}
h1{font-size:34px;letter-spacing:-.5px;margin:0 0 8px}
h2{font-size:22px;margin:44px 0 12px;padding-top:8px;border-top:1px solid var(--border)}
h3{font-size:16px;margin:26px 0 8px}
p{color:var(--text)}
.lead{color:var(--muted);font-size:17px}
code{background:var(--code);border:1px solid var(--border);padding:2px 6px;border-radius:6px;font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:13px}
pre{background:var(--code);border:1px solid var(--border);border-radius:12px;padding:16px;overflow:auto}
pre code{background:none;border:none;padding:0}
.method{display:inline-block;font-family:ui-monospace,monospace;font-size:12px;font-weight:700;padding:2px 8px;border-radius:6px;margin-right:8px;vertical-align:middle}
.get{background:rgba(35,165,90,.15);color:#3fcf78}
.put{background:rgba(124,92,255,.15);color:#9d86ff}
.patch{background:rgba(240,178,50,.15);color:#f0b232}
.del{background:rgba(242,63,67,.15);color:#f2585b}
.endpoint{font-family:ui-monospace,monospace;font-size:14px}
.card{background:var(--panel);border:1px solid var(--border);border-radius:12px;padding:16px 18px;margin:14px 0}
table{width:100%;border-collapse:collapse;margin:12px 0;font-size:14px}
th,td{text-align:left;padding:8px 10px;border-bottom:1px solid var(--border)}
th{color:var(--muted);font-weight:600}
.pill{display:inline-block;background:var(--panel2);border:1px solid var(--border);border-radius:999px;padding:2px 10px;font-size:12px;color:var(--muted)}
footer{color:#6b7079;font-size:13px;margin-top:60px;border-top:1px solid var(--border);padding-top:18px}
</style>
</head>
<body>
<div class="layout">
<aside>
  <div class="brand">zRuvix</div>
  <div class="tag">Discord presence API</div>
  <nav>
    <a href="#intro">Introduction</a>
    <div class="group">REST API</div>
    <a href="#get-user">Get presence</a>
    <a href="#me">Get @me</a>
    <a href="#card">Status card (SVG)</a>
    <a href="#history">History</a>
    <a href="#stats">Stats</a>
    <a href="#avatar">Avatar quicklink</a>
    <div class="group">Now Playing</div>
    <a href="#nowplaying">now_playing</a>
    <a href="#music">Spotify &amp; YouTube Music</a>
    <div class="group">KV Store</div>
    <a href="#kv">Reading &amp; writing</a>
    <a href="#apikey">API keys</a>
    <div class="group">Realtime</div>
    <a href="#socket">WebSocket</a>
    <div class="group">Bot</div>
    <a href="#commands">Commands</a>
    <div class="group">About</div>
    <a href="#changelog">Changelog</a>
  </nav>
</aside>
<main>
  <h1 id="intro">zRuvix API</h1>
  <p class="lead">Expose your live Discord presence and activities as a REST API and WebSocket, with a realtime key/value store. zRuvix is a Go service compatible with the Lanyard API shape.</p>
  <p><span class="pill">Base URL</span> &nbsp; <code>{{BASE}}</code> &nbsp; <span class="pill">v{{VERSION}}</span></p>

  <h2 id="get-user">Get presence</h2>
  <p><span class="method get">GET</span><span class="endpoint">/v1/users/:user_id</span></p>
  <p>Returns the full presence object for a monitored user (someone who shares a server with the bot).</p>
  <pre><code>curl {{BASE}}/v1/users/94490510688792576</code></pre>
  <p>Response (trimmed):</p>
  <pre><code>{
  "success": true,
  "data": {
    "discord_user": { "id": "...", "username": "..." },
    "discord_status": "online",
    "active_on_discord_desktop": true,
    "listening_to_spotify": true,
    "spotify": { ... },
    "listening_to_youtube_music": false,
    "youtube_music": null,
    "now_playing": { ... },
    "activities": [ ... ],
    "kv": { "location": "Tokyo" }
  }
}</code></pre>
  <div class="card">Reads are public and unauthenticated by design. If the user isn't monitored you get <code>404 user_not_monitored</code>.</div>

  <h2 id="me">Get @me</h2>
  <p><span class="method get">GET</span><span class="endpoint">/v1/users/@me</span></p>
  <p>Returns your own presence, resolved from your API key.</p>
  <pre><code>curl {{BASE}}/v1/users/@me -H "Authorization: &lt;your_api_key&gt;"</code></pre>

  <h2 id="card">Status card (SVG)</h2>
  <p><span class="method get">GET</span><span class="endpoint">/v1/users/:user_id/card.svg</span></p>
  <p>A live, self-contained, animated SVG card — drop it straight into a GitHub README or any webpage. Updates every time it's loaded.</p>
  <pre><code>![status]({{BASE}}/v1/users/94490510688792576/card.svg)</code></pre>
  <p>Shows the avatar, a pulsing status dot, and the current song or activity.</p>

  <h2 id="history">History</h2>
  <p><span class="method get">GET</span><span class="endpoint">/v1/users/:user_id/history</span></p>
  <p>Recent status changes and listening history (track changes), most-recent first. Works even when the user is offline.</p>
  <pre><code>{
  "success": true,
  "data": {
    "status_history": [ { "status": "online", "ts": 1700000000000 } ],
    "track_history":  [ { "source": "spotify", "song": "...", "artist": "...", "ts": 1700000000000 } ]
  }
}</code></pre>

  <h2 id="stats">Stats</h2>
  <p><span class="method get">GET</span><span class="endpoint">/v1/users/:user_id/stats</span></p>
  <p>Aggregate statistics for a user.</p>
  <pre><code>{
  "success": true,
  "data": {
    "current_status": "online",
    "last_seen": 1700000000000,
    "tracks_today": 12,
    "track_history_count": 47,
    "is_monitored": true
  }
}</code></pre>

  <h2 id="avatar">Avatar quicklink</h2>
  <p><span class="method get">GET</span><span class="endpoint">/:user_id.:ext</span></p>
  <p>Proxies a user's Discord avatar. <code>ext</code> is one of <code>png</code>, <code>gif</code>, <code>webp</code>, <code>jpg</code>, <code>jpeg</code>.</p>
  <pre><code>{{BASE}}/94490510688792576.png</code></pre>

  <h2 id="nowplaying">now_playing</h2>
  <p>A single, source-agnostic music object that normalizes whatever the user is listening to. <code>null</code> when nothing is playing.</p>
  <table>
    <tr><th>Field</th><th>Description</th></tr>
    <tr><td><code>source</code></td><td><code>spotify</code> or <code>youtube_music</code></td></tr>
    <tr><td><code>song</code> / <code>artist</code> / <code>album</code></td><td>Track metadata</td></tr>
    <tr><td><code>album_art_url</code></td><td>Direct image URL</td></tr>
    <tr><td><code>track_id</code> / <code>track_url</code></td><td>Track identifier and link</td></tr>
    <tr><td><code>timestamps</code> / <code>duration_ms</code></td><td>Start/end (ms) and total duration; compute live progress as <code>(now - start) / duration_ms</code></td></tr>
  </table>

  <h2 id="music">Spotify &amp; YouTube Music</h2>
  <p>Alongside <code>now_playing</code>, dedicated fields are provided for each source (matching Lanyard for Spotify):</p>
  <ul>
    <li><code>listening_to_spotify</code> (bool) &amp; <code>spotify</code> (object|null) — detected by activity id <code>spotify:1</code>.</li>
    <li><code>listening_to_youtube_music</code> (bool) &amp; <code>youtube_music</code> (object|null) — detected by activity name <code>YouTube Music</code>.</li>
  </ul>
  <div class="card">Note: the raw activity also remains in the <code>activities</code> array, so each appears both as a dedicated field and as a normal activity.</div>

  <h2 id="kv">KV store</h2>
  <p>Attach custom string data to your profile. It appears under <code>kv</code> in your presence and pushes a realtime update over the socket on change.</p>
  <p><span class="method put">PUT</span><span class="endpoint">/v1/users/:id/kv/:key</span> &nbsp; set one key (value = request body)</p>
  <p><span class="method patch">PATCH</span><span class="endpoint">/v1/users/:id/kv</span> &nbsp; merge a JSON object of pairs</p>
  <p><span class="method del">DELETE</span><span class="endpoint">/v1/users/:id/kv/:key</span> &nbsp; delete one key</p>
  <pre><code>curl -X PUT {{BASE}}/v1/users/&lt;id&gt;/kv/location \
  -H "Authorization: &lt;your_api_key&gt;" \
  -d "Tokyo"</code></pre>
  <p>Limits: keys are alphanumeric/underscore (≤255 chars), values ≤30,000 chars, up to 512 keys per user.</p>

  <h2 id="apikey">API keys</h2>
  <p>DM the bot <code>{{PREFIX}}apikey</code> to receive your secret key. Send it in the <code>Authorization</code> header for KV writes and <code>/@me</code>. Keep it private — anyone with it can edit your data.</p>

  <h2 id="socket">WebSocket</h2>
  <p><span class="method get">WSS</span><span class="endpoint">/socket</span> &nbsp; (append <code>?compression=zlib_json</code> for zlib)</p>
  <p>On connect you receive <code>Opcode 1: Hello</code> with a <code>heartbeat_interval</code>. Send <code>Opcode 2: Initialize</code> immediately, then <code>Opcode 3: Heartbeat</code> on the interval.</p>
  <table>
    <tr><th>Opcode</th><th>Name</th><th>Direction</th></tr>
    <tr><td>0</td><td>Event (INIT_STATE, PRESENCE_UPDATE)</td><td>Receive</td></tr>
    <tr><td>1</td><td>Hello</td><td>Receive</td></tr>
    <tr><td>2</td><td>Initialize</td><td>Send</td></tr>
    <tr><td>3</td><td>Heartbeat</td><td>Send</td></tr>
    <tr><td>4</td><td>Unsubscribe</td><td>Send</td></tr>
  </table>
  <pre><code>// Opcode 2 — subscribe to one, many, or all
{ "op": 2, "d": { "subscribe_to_id": "94490510688792576" } }
{ "op": 2, "d": { "subscribe_to_ids": ["id1","id2"] } }
{ "op": 2, "d": { "subscribe_to_all": true } }</code></pre>

  <h2 id="commands">Bot commands</h2>
  <p>Prefix: <code>{{PREFIX}}</code></p>
  <table>
    <tr><th>Command</th><th>What it does</th></tr>
    <tr><td><code>{{PREFIX}}set &lt;label&gt; &lt;value&gt;</code></td><td>Save a piece of info</td></tr>
    <tr><td><code>{{PREFIX}}get &lt;label&gt;</code></td><td>Read it back</td></tr>
    <tr><td><code>{{PREFIX}}del &lt;label&gt;</code></td><td>Remove one</td></tr>
    <tr><td><code>{{PREFIX}}list</code></td><td>Show everything you saved</td></tr>
    <tr><td><code>{{PREFIX}}count</code></td><td>How many you've saved</td></tr>
    <tr><td><code>{{PREFIX}}clear confirm</code></td><td>Remove everything</td></tr>
    <tr><td><code>{{PREFIX}}me</code></td><td>Show your live presence</td></tr>
    <tr><td><code>{{PREFIX}}stats</code> / <code>{{PREFIX}}ping</code></td><td>Service stats / health</td></tr>
    <tr><td><code>{{PREFIX}}apikey</code></td><td>Get your API key (DM only)</td></tr>
    <tr><td><code>{{PREFIX}}help</code></td><td>Show all commands</td></tr>
  </table>

  <h2 id="changelog">Changelog</h2>
  <p>Current version: <span class="pill">v{{VERSION}}</span> · machine-readable at <code><span class="method get">GET</span> /v1/version</code></p>
  {{CHANGELOG}}

  <footer>zRuvix — a Go service for Discord presence. Built on the Lanyard API shape.</footer>
</main>
</div>
</body>
</html>`
