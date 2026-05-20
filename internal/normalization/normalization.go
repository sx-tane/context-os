package normalization

import (
	"strings"
	"time"

	"github.com/sx-tane/context-os/shared/events"
	"github.com/sx-tane/context-os/shared/types"
)

func Normalize(event events.Event) types.NormalizedDocument {
	metadata := map[string]string{}
	for key, value := range event.Metadata {
		metadata[key] = value
	}
	return types.NormalizedDocument{
		ID:           event.ID,
		Source:       event.Source,
		SourceType:   string(event.Type),
		Title:        strings.TrimSpace(event.Subject),
		Body:         strings.TrimSpace(event.Content),
		Metadata:     metadata,
		NormalizedAt: time.Now().UTC(),
	}
}
