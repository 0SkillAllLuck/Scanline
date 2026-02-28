package timeline

import "context"

// Scrobble marks an item as watched.
//
// The ratingKey is the unique identifier for the item.
func (t *Timeline) Scrobble(ctx context.Context, ratingKey string) error {
	query := map[string]string{
		"identifier": "com.plexapp.plugins.library",
		"key":        ratingKey,
	}
	_, err := t.PutWithQuery(ctx, "/:/scrobble", query).Do()
	return err
}
