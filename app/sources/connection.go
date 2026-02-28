package sources

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sort"
	"time"

	"github.com/0skillallluck/scanline/provider/plex/auth"
)

func connectionPenalty(c auth.ResourceConnection) int {
	if c.Relay {
		return 4
	}
	if c.Local && !c.IPv6 {
		return 0
	}
	if c.Local && c.IPv6 {
		return 1
	}
	if !c.Local && !c.IPv6 {
		return 2
	}
	return 3
}

func newConnectionClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
}

func findBestConnection(ctx context.Context, httpClient *http.Client, connections []auth.ResourceConnection, token, clientIdentifier string) (string, error) {
	sorted := make([]auth.ResourceConnection, len(connections))
	copy(sorted, connections)
	sort.SliceStable(sorted, func(i, j int) bool {
		return connectionPenalty(sorted[i]) < connectionPenalty(sorted[j])
	})

	for _, conn := range sorted {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, conn.URI+"/identity", nil)
		if err != nil {
			slog.Debug("plex: connection attempt failed to create request", "uri", conn.URI, "error", err)
			continue
		}
		req.Header.Set("X-Plex-Token", token)
		req.Header.Set("X-Plex-Client-Identifier", clientIdentifier)

		resp, err := httpClient.Do(req)
		if err != nil {
			slog.Debug("plex: connection attempt failed", "uri", conn.URI, "local", conn.Local, "relay", conn.Relay, "ipv6", conn.IPv6, "error", err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			slog.Debug("plex: connection selected", "uri", conn.URI, "local", conn.Local, "relay", conn.Relay, "ipv6", conn.IPv6)
			return conn.URI, nil
		}
		slog.Debug("plex: connection attempt rejected", "uri", conn.URI, "local", conn.Local, "relay", conn.Relay, "ipv6", conn.IPv6, "status", resp.StatusCode)
	}

	return "", fmt.Errorf("no reachable connection found")
}
