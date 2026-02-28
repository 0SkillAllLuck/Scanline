package notifications

import "github.com/0skillallluck/scanline/internal/signals"

var OnToast = signals.NewStatelessSignal[string]()
