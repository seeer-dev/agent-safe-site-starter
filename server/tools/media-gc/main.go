package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"github.com/example/ai-site-starter/server/internal/config"
	"github.com/example/ai-site-starter/server/internal/modules/media"
	"github.com/example/ai-site-starter/server/internal/platform/database"
	"github.com/example/ai-site-starter/server/internal/platform/storage"
)

func main() {
	apply := flag.Bool("apply", false, "claim eligible media and delete it from object storage")
	batch := flag.Int("batch", 100, "maximum number of candidates and pending jobs to process")
	flag.Parse()
	if *batch <= 0 || *batch > 1000 {
		log.Fatal("--batch must be between 1 and 1000")
	}

	ctx := context.Background()
	cfg := config.Load()
	db, dialect, err := database.Open(ctx, cfg.DBDriver, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	registry := media.NewSQLRegistryStore(db, dialect)

	var summary media.GCSummary
	if *apply {
		if !cfg.R2Enabled() {
			log.Fatal("R2 credentials and bucket are required with --apply")
		}
		r2, err := storage.NewR2(ctx, cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2Bucket)
		if err != nil {
			log.Fatalf("initialize R2: %v", err)
		}
		summary, err = media.NewCollector(registry, r2).Collect(ctx, time.Now(), *batch)
		if err != nil {
			log.Fatalf("media gc: %v", err)
		}
	} else {
		summary, err = media.NewCollector(registry, nil).Preview(ctx, time.Now(), *batch)
		if err != nil {
			log.Fatalf("media gc preview: %v", err)
		}
	}

	if err := json.NewEncoder(os.Stdout).Encode(summary); err != nil {
		log.Fatalf("encode summary: %v", err)
	}
	if summary.Failed > 0 {
		log.Fatalf("%d media deletion jobs remain retryable after provider failures", summary.Failed)
	}
}
