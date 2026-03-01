package sources

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
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

const connectionTimeout = 3 * time.Second

func newConnectionClient() *http.Client {
	return &http.Client{
		Timeout: connectionTimeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: connectionTimeout,
			}).DialContext,
			TLSHandshakeTimeout: connectionTimeout,
		},
	}
}

// connResult holds the outcome of a single connection probe.
type connResult struct {
	uri     string
	penalty int
	ok      bool
}

func findBestConnection(ctx context.Context, httpClient *http.Client, connections []auth.ResourceConnection, token, clientIdentifier string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	results := make([]connResult, len(connections))
	var wg sync.WaitGroup
	for i, conn := range connections {
		wg.Add(1)
		go func(i int, conn auth.ResourceConnection) {
			defer wg.Done()
			results[i] = connResult{uri: conn.URI, penalty: connectionPenalty(conn)}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, conn.URI+"/identity", nil)
			if err != nil {
				slog.Debug("plex: connection attempt failed to create request", "uri", conn.URI, "error", err)
				return
			}
			req.Header.Set("X-Plex-Token", token)
			req.Header.Set("X-Plex-Client-Identifier", clientIdentifier)

			resp, err := httpClient.Do(req)
			if err != nil {
				slog.Debug("plex: connection attempt failed", "uri", conn.URI, "local", conn.Local, "relay", conn.Relay, "ipv6", conn.IPv6, "error", err)
				return
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				slog.Debug("plex: connection reachable", "uri", conn.URI, "local", conn.Local, "relay", conn.Relay, "ipv6", conn.IPv6)
				results[i].ok = true
			} else {
				slog.Debug("plex: connection attempt rejected", "uri", conn.URI, "local", conn.Local, "relay", conn.Relay, "ipv6", conn.IPv6, "status", resp.StatusCode)
			}
		}(i, conn)
	}
	wg.Wait()

	// Pick the reachable connection with the lowest penalty.
	best := -1
	for i, r := range results {
		if r.ok && (best == -1 || r.penalty < results[best].penalty) {
			best = i
		}
	}
	if best == -1 {
		return "", fmt.Errorf("no reachable connection found")
	}

	slog.Debug("plex: connection selected", "uri", results[best].uri)
	return results[best].uri, nil
}
