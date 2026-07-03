package service

import (
	"sync"
	"time"

	"github.com/eugegm01-dev/shortener/internal/storage"
	"github.com/eugegm01-dev/shortener/pkg/logger"
)

type DeleteJob struct {
	UserID   string
	ShortIDs []string
}

type Deleter struct {
	storage   storage.Storage
	jobs      chan DeleteJob
	quit      chan struct{}
	wg        sync.WaitGroup
	batchSize int
	timeout   time.Duration
}

func NewDeleter(s storage.Storage, batchSize int, timeout time.Duration) *Deleter {
	d := &Deleter{
		storage:   s,
		jobs:      make(chan DeleteJob, 10000),
		quit:      make(chan struct{}),
		batchSize: batchSize,
		timeout:   timeout,
	}
	d.wg.Add(1)
	go d.run()
	return d
}

func (d *Deleter) Enqueue(userID string, ids []string) {
	d.jobs <- DeleteJob{UserID: userID, ShortIDs: ids}
}

func (d *Deleter) run() {
	defer d.wg.Done()
	buffer := make(map[string][]string)
	totalIDs := 0
	ticker := time.NewTicker(d.timeout)
	defer ticker.Stop()

	flush := func() {
		if totalIDs == 0 {
			return
		}
		for uid, ids := range buffer {
			if err := d.storage.DeleteUserURLs(uid, ids); err != nil {
				logger.Logger.Error().Err(err).Str("user_id", uid).Int("count", len(ids)).Msg("batch delete failed")
			}
		}
		buffer = make(map[string][]string)
		totalIDs = 0
	}

	for {
		select {
		case job := <-d.jobs:
			buffer[job.UserID] = append(buffer[job.UserID], job.ShortIDs...)
			totalIDs += len(job.ShortIDs)
			if totalIDs >= d.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-d.quit:
			for {
				select {
				case job := <-d.jobs:
					buffer[job.UserID] = append(buffer[job.UserID], job.ShortIDs...)
				default:
					flush()
					return
				}
			}
		}
	}
}

func (d *Deleter) Close() {
	close(d.quit)
	d.wg.Wait()
}
