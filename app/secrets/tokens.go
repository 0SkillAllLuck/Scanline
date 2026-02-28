package secrets

import "log/slog"

// GetToken returns the value for a generic keyring key, or empty string on error.
func GetToken(key string) string {
	item, err := getService().Get(key)
	if err != nil {
		return ""
	}
	return item.Password
}

// SetToken stores a value under a generic keyring key.
func SetToken(key, token string) {
	err := getService().Set(key, Item{Label: "Scanline: " + key, Password: token})
	if err != nil {
		slog.Error("failed to set keyring token", "key", key, "error", err)
	}
}

// DeleteToken removes a generic keyring key.
func DeleteToken(key string) {
	err := getService().Delete(key)
	if err != nil {
		slog.Debug("failed to delete keyring key (may not exist)", "key", key, "error", err)
	}
}

// HasToken returns true if a generic keyring key exists.
func HasToken(key string) bool {
	exists, err := getService().Has(key)
	if err != nil {
		return false
	}
	return exists
}
