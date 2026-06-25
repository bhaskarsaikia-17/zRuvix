// Package metrics defines the Prometheus collectors used across the service,
// mirroring zRuvix.Metrics.Collector from the Elixir implementation.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"net/http"
)

var (
	ConnectedSessions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "zruvix_connected_sessions",
		Help: "Currently connected sessions count.",
	})
	MessagesOutbound = promauto.NewCounter(prometheus.CounterOpts{
		Name: "zruvix_messages_outbound",
		Help: "Total socket messages outbound.",
	})
	MessagesInbound = promauto.NewCounter(prometheus.CounterOpts{
		Name: "zruvix_messages_inbound",
		Help: "Total messages received count.",
	})
	PresenceUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Name: "zruvix_presence_updates",
		Help: "Presence updates received count.",
	})
	MonitoredUsers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "zruvix_monitored_users",
		Help: "Users monitored by zRuvix count.",
	})
	Responses2xx = promauto.NewCounter(prometheus.CounterOpts{
		Name: "zruvix_2xx_responses",
		Help: "2xx http responses",
	})
	Responses4xx = promauto.NewCounter(prometheus.CounterOpts{
		Name: "zruvix_4xx_responses",
		Help: "4xx http responses",
	})
	Responses5xx = promauto.NewCounter(prometheus.CounterOpts{
		Name: "zruvix_5xx_responses",
		Help: "5xx http responses",
	})
	DiscordMessagesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "zruvix_discord_messages_sent",
		Help: "Messages sent to discord count",
	})
)

// Handler returns the Prometheus scrape handler for the /metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}
