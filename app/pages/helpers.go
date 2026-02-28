package pages

import "fmt"

func errSourceNotFound(serverID string) error {
	return fmt.Errorf("source not found: %s", serverID)
}
