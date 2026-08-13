package service

import (
	"context"
	"time"

	"github.com/gradis/ya-pr_shorturl/internal/repository"
	"go.uber.org/zap"
)

const (
	deleteQueueSize     = 100
	deleteBatchSize     = 100
	deleteFlushInterval = 100 * time.Millisecond
	deleteQueryTimeout  = 5 * time.Second
)

type deleteRequest struct {
	UserID string
	IDs    []string
}

func (s *URLService) startDeleteWorker() {
	s.deleteWG.Add(1)

	go func() {
		defer s.deleteWG.Done()

		ticker := time.NewTicker(deleteFlushInterval)
		defer ticker.Stop()

		batch := make(
			[]repository.URLDeleteRecord,
			0,
			deleteBatchSize,
		)

		for {
			select {
			case request, ok := <-s.deleteQueue:
				if !ok {
					batch = s.flushDeleteBatch(batch)
					return
				}

				for _, id := range request.IDs {
					batch = append(
						batch,
						repository.URLDeleteRecord{
							ID:     id,
							UserID: request.UserID,
						},
					)

					if len(batch) >= deleteBatchSize {
						batch = s.flushDeleteBatch(batch)
					}
				}

			case <-ticker.C:
				batch = s.flushDeleteBatch(batch)
			}
		}
	}()
}

func (s *URLService) flushDeleteBatch(
	batch []repository.URLDeleteRecord,
) []repository.URLDeleteRecord {
	if len(batch) == 0 {
		return batch
	}

	records := append(
		[]repository.URLDeleteRecord(nil),
		batch...,
	)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		deleteQueryTimeout,
	)
	defer cancel()

	if err := s.repo.DeleteBatch(ctx, records); err != nil {
		s.logg.Error(
			"failed to delete URL batch",
			zap.Int("batch_size", len(records)),
			zap.Error(err),
		)
	}

	return batch[:0]
}

func (s *URLService) Close() {
	s.closeOnce.Do(func() {
		close(s.deleteQueue)
		s.deleteWG.Wait()
	})
}
