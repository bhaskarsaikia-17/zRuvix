// Package metrics defines the Prometheus collectors used across the service,
// mirroring Lanyard.Metrics.Collector from the Elixir implementation.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"net/http"
)

var (
	ConnectedSessions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "lanyard_connected_sessions",
		Help: "Currently connected sessions count.",
	})
	MessagesOutbound = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lanyard_messages_outbound",
		Help: "Total socket messages outbound.",
	})
	MessagesInbound = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lanyard_messages_inbound",
		Help: "Total messages received count.",
	})
	PresenceUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lanyard_presence_updates",
		Help: "Presence updates received count.",
	})
	MonitoredUsers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "lanyard_monitored_users",
		Help: "Users monitored by Lanyard count.",
	})
	Responses2xx = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lanyard_2xx_responses",
		Help: "2xx http responses",
	})
	Responses4xx = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lanyard_4xx_responses",
		Help: "4xx http responses",
	})
	Responses5xx = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lanyard_5xx_responses",
		Help: "5xx http responses",
	})
	DiscordMessagesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lanyard_discord_messages_sent",
		Help: "Messages sent to discord count",
	})
)

// Handler returns the Prometheus scrape handler for the /metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}
