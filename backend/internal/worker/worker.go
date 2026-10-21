// Package worker runs periodic background jobs with clean shutdown and heartbeats in worker_status.
package worker

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"

	"github.com/g-asiapacific/elm/backend/internal/store"
)

type Job struct {
	Name     string
	Interval time.Duration
	Timeout  time.Duration
	Run      func(ctx context.Context) error
}

// Start launches one goroutine per job. Each run happens immediately and then every Interval.
// The returned wait function blocks until all jobs have stopped after ctx is cancelled.
func Start(ctx context.Context, db *sql.DB, log *slog.Logger, jobs ...Job) (wait func()) {
	q := store.New(db)
	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			t := time.NewTicker(j.Interval)
			defer t.Stop()
			for {
				runOnce(ctx, q, log, j)
				select {
				case <-ctx.Done():
					return
				case <-t.C:
				}
			}
		}()
	}
	return wg.Wait
}

func runOnce(ctx context.Context, q *store.Queries, log *slog.Logger, j Job) {
	timeout := j.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now().UTC()
	err := j.Run(runCtx)
	rec := store.RecordWorkerRunParams{Name: j.Name, RunAt: sql.NullTime{Time: start, Valid: true}}
	if err != nil {
		if ctx.Err() != nil {
			return // shutting down
		}
		log.Error("worker job failed", slog.String("job", j.Name), slog.String("error", err.Error()))
		// Details stay in the server log; the database (shown on /admin/system) gets a safe message only.
		rec.LastError = sql.NullString{String: "Job failed. See server logs.", Valid: true}
		logCtx, cancelLog := context.WithTimeout(context.Background(), 5*time.Second)
		_ = q.InsertOperationalError(logCtx, store.InsertOperationalErrorParams{Component: j.Name, Code: "job_failed", Message: "Background job failed. See server logs for details."})
		cancelLog()
	} else {
		rec.SuccessAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
	}
	hbCtx, hbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer hbCancel()
	if err := q.RecordWorkerRun(hbCtx, rec); err != nil {
		log.Warn("worker heartbeat failed", slog.String("job", j.Name), slog.String("error", err.Error()))
	}
}
