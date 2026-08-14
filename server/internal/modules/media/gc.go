package media

import (
	"context"
	"fmt"
	"time"

	"github.com/example/ai-site-starter/server/internal/platform/storage"
)

const (
	verifiedMediaRetention     = 7 * 24 * time.Hour
	staleVerificationRetention = 24 * time.Hour
)

// GCSummary intentionally contains counts only. Object keys can contain user
// identifiers and are not emitted by the operational tool.
type GCSummary struct {
	DryRun   bool `json:"dry_run"`
	Eligible int  `json:"eligible"`
	Claimed  int  `json:"claimed"`
	Deleted  int  `json:"deleted"`
	Failed   int  `json:"failed"`
	Pending  int  `json:"pending"`
}

// Collector coordinates durable database claims with idempotent object-store
// deletion. A provider failure leaves the job in media_gc_jobs for retry.
type Collector struct {
	store       GCStore
	objectStore storage.Store
}

func NewCollector(store GCStore, objectStore storage.Store) Collector {
	return Collector{store: store, objectStore: objectStore}
}

func (c Collector) Preview(ctx context.Context, now time.Time, limit int) (GCSummary, error) {
	keys, err := c.store.ListEligible(ctx, now.Unix(), limit)
	if err != nil {
		return GCSummary{}, err
	}
	jobs, err := c.store.ListGCJobs(ctx, limit)
	if err != nil {
		return GCSummary{}, err
	}
	return GCSummary{DryRun: true, Eligible: len(keys), Pending: len(jobs)}, nil
}

func (c Collector) Collect(ctx context.Context, now time.Time, limit int) (GCSummary, error) {
	if c.objectStore == nil {
		return GCSummary{}, fmt.Errorf("media gc object store is required")
	}
	claimed, err := c.store.ClaimEligible(ctx, now.Unix(), limit)
	if err != nil {
		return GCSummary{}, err
	}
	jobs, err := c.store.ListGCJobs(ctx, limit)
	if err != nil {
		return GCSummary{}, err
	}
	summary := GCSummary{Claimed: len(claimed), Pending: len(jobs)}
	for _, job := range jobs {
		if err := c.objectStore.DeleteObject(ctx, job.ObjectKey); err != nil {
			summary.Failed++
			if markErr := c.store.MarkGCFailed(ctx, job.ObjectKey, now.Unix()); markErr != nil {
				return summary, fmt.Errorf("delete media object failed and retry state could not be recorded: %w", markErr)
			}
			continue
		}
		if err := c.store.MarkGCSucceeded(ctx, job.ObjectKey); err != nil {
			return summary, err
		}
		summary.Deleted++
		summary.Pending--
	}
	return summary, nil
}
