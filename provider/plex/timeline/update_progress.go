package timeline

import (
	"context"
	"strconv"
)

// UpdateProgress updates the playback progress for an item.
//
// The ratingKey is the unique identifier for the item.
// The state should be one of StatePlaying, StatePaused, or StateStopped.
// The time is the current playback position in milliseconds.
// The duration is the total duration in milliseconds.
func (t *Timeline) UpdateProgress(ctx context.Context, ratingKey string, state PlaybackState, time, duration int) error {
	query := map[string]string{
		"ratingKey": ratingKey,
		"state":     string(state),
		"time":      strconv.Itoa(time),
		"duration":  strconv.Itoa(duration),
		"key":       "/library/metadata/" + ratingKey,
	}
	_, err := t.Request("POST", "/:/timeline").
		WithContext(ctx).
		WithQuery(query).
		Do()
	return err
}
