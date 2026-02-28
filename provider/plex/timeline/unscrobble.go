package timeline

import "context"

// Unscrobble marks an item as unwatched.
//
// The ratingKey is the unique identifier for the item.
func (t *Timeline) Unscrobble(ctx context.Context, ratingKey string) error {
	query := map[string]string{
		"identifier": "com.plexapp.plugins.library",
		"key":        ratingKey,
	}
	_, err := t.PutWithQuery(ctx, "/:/unscrobble", query).Do()
	return err
}
