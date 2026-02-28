package secrets

import (
	"log/slog"

	"github.com/google/uuid"
)

const plexClientIDKey = "plex_client_id"

func GetOrCreateClientID() string {
	exists, err := getService().Has(plexClientIDKey)
	if err != nil {
		slog.Error("error checking client ID in keyring", "error", err)
		return uuid.New().String()
	}

	if exists {
		item, err := getService().Get(plexClientIDKey)
		if err != nil {
			slog.Error("error reading client ID from keyring", "error", err)
			return uuid.New().String()
		}
		return item.Password
	}

	id := uuid.New().String()
	err = getService().Set(plexClientIDKey, Item{Label: "Scanline Plex Client ID", Password: id})
	if err != nil {
		slog.Error("error storing client ID in keyring", "error", err)
	}
	return id
}
