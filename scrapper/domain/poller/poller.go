package poller

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/botclient"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/infra/external"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/scrapper/repository"
)

type Config struct {
	DBPageSize       int
	SuperBatchSize   int
	WorkerCount      int
	CheckInterval    time.Duration
}

type Poller struct {
	store  repository.TrackedLinkService
	ext    external.LinkChangesClient
	sender botclient.MessageSender
	cfg    Config
}

func New(
	store repository.TrackedLinkService,
	ext external.LinkChangesClient,
	sender botclient.MessageSender,
	cfg Config,
) *Poller {
	if cfg.WorkerCount < 1 {
		cfg.WorkerCount = 1
	}
	if cfg.DBPageSize < 1 {
		cfg.DBPageSize = 100
	}
	if cfg.SuperBatchSize < 1 {
		cfg.SuperBatchSize = 1000
	}
	return &Poller{store: store, ext: ext, sender: sender, cfg: cfg}
}

func (p *Poller) ProcessOnce(ctx context.Context) {
	checkedBefore := time.Now().UTC().Add(-p.cfg.CheckInterval)
	var afterID int64
	for {
		batch, lastID, err := p.collectSuperBatch(ctx, checkedBefore, afterID)
		if err != nil {
			slog.Error("Failed to load tracked links batch", slog.String("error", err.Error()))
			return
		}
		if len(batch) == 0 {
			return
		}
		afterID = lastID
		failedByChat := p.processBatchParallel(ctx, batch)
		p.sendFailureReports(ctx, failedByChat)
	}
}

func (p *Poller) collectSuperBatch(ctx context.Context, checkedBefore time.Time, afterID int64) ([]repository.TrackedLink, int64, error) {
	_ = ctx
	const minCap = 16
	out := make([]repository.TrackedLink, 0, minCap)
	cursor := afterID
	for len(out) < p.cfg.SuperBatchSize {
		page, err := p.store.ListTrackedLinksPage(p.cfg.DBPageSize, checkedBefore, cursor)
		if err != nil {
			return nil, 0, err
		}
		if len(page) == 0 {
			break
		}
		out = append(out, page...)
		cursor = page[len(page)-1].Response.ID
		if len(page) < p.cfg.DBPageSize {
			break
		}
	}
	if len(out) == 0 {
		return nil, afterID, nil
	}
	return out, cursor, nil
}

func (p *Poller) processBatchParallel(ctx context.Context, batch []repository.TrackedLink) map[int64][]string {
	chunks := splitIntoChunks(batch, p.cfg.WorkerCount)
	var mu sync.Mutex
	failed := make(map[int64][]string)
	var wg sync.WaitGroup
	for _, chunk := range chunks {
		if len(chunk) == 0 {
			continue
		}
		wg.Add(1)
		go func(links []repository.TrackedLink) {
			defer wg.Done()
			for _, link := range links {
				p.processOneLink(ctx, link, &mu, failed)
			}
		}(chunk)
	}
	wg.Wait()
	return failed
}

func (p *Poller) processOneLink(
	ctx context.Context,
	link repository.TrackedLink,
	mu *sync.Mutex,
	failed map[int64][]string,
) {
	logger := slog.With("url", link.Response.URL)
	changes, wm, fetchErr := p.ext.FetchChangesSince(ctx, link.Response.URL, link.LastUpdated)
	if fetchErr != nil {
		logger.Warn("Failed to fetch changes from external source", slog.String("error", fetchErr.Error()))
		mu.Lock()
		for chatID := range link.ChatIDs {
			failed[chatID] = append(failed[chatID], link.Response.URL)
		}
		mu.Unlock()
		if touchErr := p.store.TouchLastChecked(link.Response.URL, time.Now().UTC()); touchErr != nil {
			logger.Warn("Failed to mark last_checked_at after external error", slog.String("error", touchErr.Error()))
		}
		return
	}

	chatIDs := chatIDsSorted(link.ChatIDs)

	if len(changes) == 0 {
		if !wm.IsZero() && (link.LastUpdated.IsZero() || wm.After(link.LastUpdated)) {
			if updErr := p.store.UpdateLastUpdated(link.Response.URL, wm); updErr != nil {
				logger.Warn("Failed to update baseline last_updated", slog.String("error", updErr.Error()))
			}
		}
		if touchErr := p.store.TouchLastChecked(link.Response.URL, time.Now().UTC()); touchErr != nil {
			logger.Warn("Failed to mark last_checked_at", slog.String("error", touchErr.Error()))
		}
		return
	}

	allSent := true
	for _, ch := range changes {
		update := trackerapi.NewLinkUpdate(
			link.Response.ID,
			link.Response.URL,
			chatIDs,
			ch.Kind,
			ch.Title,
			ch.Author,
			ch.CreatedAt,
			ch.Preview,
		)
		if sendErr := p.sender.SendLinkUpdate(ctx, update); sendErr != nil {
			logger.Warn("Failed to send update to bot", slog.String("error", sendErr.Error()))
			allSent = false
			break
		}
	}

	if allSent {
		if !wm.IsZero() {
			if updErr := p.store.UpdateLastUpdated(link.Response.URL, wm); updErr != nil {
				logger.Warn("Failed to update last_updated", slog.String("error", updErr.Error()))
			}
		}
	} else {
		mu.Lock()
		for chatID := range link.ChatIDs {
			failed[chatID] = append(failed[chatID], link.Response.URL)
		}
		mu.Unlock()
	}
	if touchErr := p.store.TouchLastChecked(link.Response.URL, time.Now().UTC()); touchErr != nil {
		logger.Warn("Failed to mark last_checked_at", slog.String("error", touchErr.Error()))
	}
}

func (p *Poller) sendFailureReports(ctx context.Context, failed map[int64][]string) {
	for chatID, urls := range failed {
		if len(urls) == 0 {
			continue
		}
		uniq := dedupeStrings(urls)
		report := trackerapi.ProcessingFailureReport{
			TgChatID: chatID,
			URLs:     uniq,
			Detail:   "Повторите попытку позже или проверьте доступность ресурса.",
		}
		if err := p.sender.SendProcessingFailureReport(ctx, report); err != nil {
			slog.Warn("Failed to send processing failure report",
				slog.String("error", err.Error()),
				slog.Int64("chat_id", chatID),
			)
		}
	}
}

func chatIDsSorted(m map[int64]struct{}) []int64 {
	out := make([]int64, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func splitIntoChunks[T any](items []T, n int) [][]T {
	if n < 1 {
		n = 1
	}
	if len(items) == 0 {
		return nil
	}
	chunks := make([][]T, n)
	base := len(items) / n
	rem := len(items) % n
	start := 0
	for i := 0; i < n; i++ {
		sz := base
		if i < rem {
			sz++
		}
		end := start + sz
		if start >= len(items) {
			chunks[i] = nil
			continue
		}
		if end > len(items) {
			end = len(items)
		}
		chunks[i] = items[start:end]
		start = end
	}
	return chunks
}
