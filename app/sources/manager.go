package sources

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/0skillallluck/scanline/app/secrets"
	"github.com/0skillallluck/scanline/internal/signals"
	"github.com/0skillallluck/scanline/provider/plex"
	"github.com/0skillallluck/scanline/provider/plex/auth"
	"github.com/google/uuid"
)

// Manager manages multi-account sources with signals for UI updates.
type Manager struct {
	accounts []*Account
	sources  map[string]Source // serverID → Source (enabled+resolved only)
	mu       sync.RWMutex

	// SourcesChanged fires when accounts, servers, or enabled state changes.
	SourcesChanged *signals.StatelessSignal[struct{}]
}

// NewManager creates a new Manager, loading config and cleaning up legacy keyring keys.
func NewManager() *Manager {
	m := &Manager{
		sources:        make(map[string]Source),
		SourcesChanged: signals.NewStatelessSignal[struct{}](),
	}

	accounts, err := loadConfig()
	if err != nil {
		slog.Error("failed to load sources config", "error", err)
	}
	if accounts != nil {
		m.accounts = accounts
	} else {
		m.accounts = []*Account{}
	}

	// Create Sources for all enabled servers, loading server tokens from keyring
	for _, acct := range m.accounts {
		token := secrets.GetToken("plex_token_" + acct.ID)
		if token == "" {
			continue
		}
		for _, srv := range acct.Servers {
			// Load server-specific access token from keyring
			if srvToken := secrets.GetToken(serverTokenKey(acct.ID, srv.ID)); srvToken != "" {
				srv.AccessToken = srvToken
			}
			if srv.Enabled && srv.URL != "" {
				client := plex.NewClient(srv.URL, tokenForServer(token, srv), acct.ClientID)
				m.sources[srv.ID] = NewPlexSource(srv.ID, srv.Name, client)
			}
		}
	}

	// Clean up legacy single-account keyring keys
	cleanupLegacyKeys()

	return m
}

// serverTokenKey returns the keyring key for a server-specific access token.
func serverTokenKey(accountID, serverID string) string {
	return "plex_server_token_" + accountID + "_" + serverID
}

// tokenForServer returns the server-specific access token if available, otherwise the account token.
func tokenForServer(accountToken string, srv *Server) string {
	if srv.AccessToken != "" {
		return srv.AccessToken
	}
	return accountToken
}

// Accounts returns all accounts.
func (m *Manager) Accounts() []*Account {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.accounts
}

// HasAccounts returns true if any accounts exist.
func (m *Manager) HasAccounts() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.accounts) > 0
}

// EnabledSources returns all active Source instances for enabled servers.
func (m *Manager) EnabledSources() []Source {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Source, 0, len(m.sources))
	for _, src := range m.sources {
		result = append(result, src)
	}
	return result
}

// SourceForServer returns the Source for a specific server ID, or nil.
func (m *Manager) SourceForServer(serverID string) Source {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sources[serverID]
}

// AddPlexAccount adds a new Plex account with discovered servers.
// Owned servers are auto-enabled and their connections resolved.
func (m *Manager) AddPlexAccount(ctx context.Context, token, username, clientID string, resources []auth.Resource) {
	accountID := uuid.New().String()

	// Keyring I/O outside lock
	secrets.SetToken("plex_token_"+accountID, token)

	// Network I/O outside lock — resolve connections for owned servers
	connClient := newConnectionClient()
	serverCount := 0
	for _, r := range resources {
		if strings.Contains(r.Provides, "server") {
			serverCount++
		}
	}
	slog.Debug("plex: adding account", "resource_count", len(resources), "server_count", serverCount)
	var servers []*Server
	for _, r := range resources {
		if !strings.Contains(r.Provides, "server") {
			continue
		}
		srv := &Server{
			ID:          r.ClientIdentifier,
			Name:        r.Name,
			Enabled:     r.Owned,
			AccessToken: r.AccessToken,
		}

		// Store server-specific access token in keyring
		if r.AccessToken != "" {
			secrets.SetToken(serverTokenKey(accountID, srv.ID), r.AccessToken)
		}

		if r.Owned {
			url, err := findBestConnection(ctx, connClient, r.Connections, token, clientID)
			if err != nil {
				slog.Warn("failed to resolve server connection", "server", r.Name, "error", err)
				srv.Enabled = false
			} else {
				slog.Debug("plex: server connection resolved", "server", r.Name, "url", url)
				srv.URL = url
			}
		}

		servers = append(servers, srv)
	}

	acct := &Account{
		ID:       accountID,
		Type:     ProviderPlex,
		Username: username,
		ClientID: clientID,
		Servers:  servers,
	}

	// Lock only for state mutation
	m.mu.Lock()
	m.accounts = append(m.accounts, acct)
	for _, srv := range servers {
		if srv.Enabled && srv.URL != "" {
			client := plex.NewClient(srv.URL, tokenForServer(token, srv), clientID)
			m.sources[srv.ID] = NewPlexSource(srv.ID, srv.Name, client)
		}
	}
	saveConfig(m.accounts)
	m.mu.Unlock()

	m.SourcesChanged.Notify(struct{}{})
}

// RemoveAccount deletes an account and its keyring tokens.
func (m *Manager) RemoveAccount(accountID string) {
	m.mu.Lock()
	var remaining []*Account
	var tokenKey string
	var serverIDs []string
	for _, acct := range m.accounts {
		if acct.ID == accountID {
			for _, srv := range acct.Servers {
				delete(m.sources, srv.ID)
				serverIDs = append(serverIDs, srv.ID)
			}
			tokenKey = "plex_token_" + acct.ID
			continue
		}
		remaining = append(remaining, acct)
	}
	m.accounts = remaining
	if m.accounts == nil {
		m.accounts = []*Account{}
	}
	saveConfig(m.accounts)
	m.mu.Unlock()

	// Keyring I/O outside lock
	if tokenKey != "" {
		secrets.DeleteToken(tokenKey)
		for _, srvID := range serverIDs {
			secrets.DeleteToken(serverTokenKey(accountID, srvID))
		}
	}
	m.SourcesChanged.Notify(struct{}{})
}

// SetServerEnabled toggles a server's enabled state.
func (m *Manager) SetServerEnabled(accountID, serverID string, enabled bool) {
	// Keyring I/O outside lock
	token := secrets.GetToken("plex_token_" + accountID)

	m.mu.Lock()
	found := false
	for _, acct := range m.accounts {
		if acct.ID != accountID {
			continue
		}
		for _, srv := range acct.Servers {
			if srv.ID != serverID {
				continue
			}
			srv.Enabled = enabled

			if enabled {
				if srv.URL == "" {
					slog.Warn("server URL not cached, need to refresh servers", "server", srv.Name)
				}
				if srv.URL != "" && token != "" {
					client := plex.NewClient(srv.URL, tokenForServer(token, srv), acct.ClientID)
					m.sources[srv.ID] = NewPlexSource(srv.ID, srv.Name, client)
				}
			} else {
				delete(m.sources, srv.ID)
			}

			found = true
			break
		}
		if found {
			break
		}
	}
	if found {
		saveConfig(m.accounts)
	}
	m.mu.Unlock()

	if found {
		m.SourcesChanged.Notify(struct{}{})
	}
}

// refreshResult holds the result of discovering and resolving servers for one account.
type refreshResult struct {
	accountID string
	servers   []*Server
	sources   map[string]Source
}

// RefreshServers re-discovers servers for all accounts from plex.tv.
func (m *Manager) RefreshServers(ctx context.Context) {
	// Snapshot account info under RLock
	m.mu.RLock()
	type accountInfo struct {
		id           string
		providerType ProviderType
		username     string
		clientID     string
		enabledState map[string]bool
	}
	infos := make([]accountInfo, 0, len(m.accounts))
	for _, acct := range m.accounts {
		if acct.Type != ProviderPlex {
			continue
		}
		enabled := make(map[string]bool)
		for _, srv := range acct.Servers {
			enabled[srv.ID] = srv.Enabled
		}
		infos = append(infos, accountInfo{
			id:           acct.ID,
			providerType: acct.Type,
			username:     acct.Username,
			clientID:     acct.ClientID,
			enabledState: enabled,
		})
	}
	m.mu.RUnlock()

	// All network/keyring I/O outside lock — parallelize across accounts
	slog.Debug("plex: refreshing servers", "account_count", len(infos))
	connClient := newConnectionClient()
	var resultsMu sync.Mutex
	var results []refreshResult
	var wg sync.WaitGroup
	for _, info := range infos {
		wg.Add(1)
		go func(info accountInfo) {
			defer wg.Done()
			token := secrets.GetToken("plex_token_" + info.id)
			if token == "" {
				return
			}

			resources, err := auth.DiscoverServers(ctx, token, info.clientID)
			if err != nil {
				slog.Warn("failed to refresh servers for account", "account", info.username, "error", err)
				return
			}
			slog.Debug("plex: refresh discovered resources", "account", info.username, "resource_count", len(resources))

			var newServers []*Server
			newSources := make(map[string]Source)
			for _, r := range resources {
				if !strings.Contains(r.Provides, "server") {
					continue
				}

				enabled, known := info.enabledState[r.ClientIdentifier]
				if !known {
					enabled = r.Owned
				}

				srv := &Server{
					ID:          r.ClientIdentifier,
					Name:        r.Name,
					Enabled:     enabled,
					AccessToken: r.AccessToken,
				}

				// Update server-specific access token in keyring
				if r.AccessToken != "" {
					secrets.SetToken(serverTokenKey(info.id, srv.ID), r.AccessToken)
				}

				if enabled {
					url, err := findBestConnection(ctx, connClient, r.Connections, token, info.clientID)
					if err != nil {
						slog.Warn("failed to resolve server connection", "server", r.Name, "error", err)
						srv.Enabled = false
					} else {
						srv.URL = url
						client := plex.NewClient(url, tokenForServer(token, srv), info.clientID)
						newSources[srv.ID] = NewPlexSource(srv.ID, srv.Name, client)
					}
				}

				newServers = append(newServers, srv)
			}

			resultsMu.Lock()
			results = append(results, refreshResult{
				accountID: info.id,
				servers:   newServers,
				sources:   newSources,
			})
			resultsMu.Unlock()
		}(info)
	}
	wg.Wait()

	// Apply results under Lock
	m.mu.Lock()
	for _, res := range results {
		for _, acct := range m.accounts {
			if acct.ID != res.accountID {
				continue
			}
			// Remove old sources for this account
			for _, srv := range acct.Servers {
				delete(m.sources, srv.ID)
			}
			acct.Servers = res.servers
			for id, src := range res.sources {
				m.sources[id] = src
			}
			break
		}
	}
	saveConfig(m.accounts)
	m.mu.Unlock()

	m.SourcesChanged.Notify(struct{}{})
}

// WindowTitle returns a suitable window title based on enabled sources.
func (m *Manager) WindowTitle() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.sources) == 1 {
		for _, src := range m.sources {
			return "Scanline - " + src.Name()
		}
	}
	return "Scanline"
}

// cleanupLegacyKeys removes old single-account keyring entries.
func cleanupLegacyKeys() {
	legacyKeys := []string{"plex_token", "plex_server_url", "plex_server_name"}
	for _, key := range legacyKeys {
		if secrets.HasToken(key) {
			secrets.DeleteToken(key)
		}
	}
}
