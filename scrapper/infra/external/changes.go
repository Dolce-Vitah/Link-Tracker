package external

import (
	"context"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/domain/linkchange"
)

type LinkChangesClient interface {
	FetchChangesSince(ctx context.Context, rawURL string, since time.Time) (changes []linkchange.Change, newWatermark time.Time, err error)
}
