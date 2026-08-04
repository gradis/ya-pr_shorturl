package service

import (
	"context"
	"strings"
	"time"

	"github.com/gradis/ya-pr_shorturl/internal/auth"
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

func (s *URLService) DeleteUserURLs(
	ctx context.Context,
	ids []string,
) error {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return ErrUnauthorized
	}

	cleanIDs := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		if _, exists := seen[id]; exists {
			continue
		}

		seen[id] = struct{}{}
		cleanIDs = append(cleanIDs, id)
	}

	if len(cleanIDs) == 0 {
		return ErrInvalidURLIDs
	}

	request := deleteRequest{
		UserID: userID,
		IDs:    cleanIDs,
	}

	select {
	case s.deleteQueue <- request:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
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

		flush := func() {
			if len(batch) == 0 {
				return
			}

			records := append(
				[]repository.URLDeleteRecord(nil),
				batch...,
			)

			batch = batch[:0]

			ctx, cancel := context.WithTimeout(
				context.Background(),
				deleteQueryTimeout,
			)

			err := s.repo.DeleteBatch(ctx, records)
			cancel()

			if err != nil {
				s.logg.Error(
					"failed to delete URL batch",
					zap.Int("batch_size", len(records)),
					zap.Error(err),
				)
			}
		}

		for {
			select {
			case request, ok := <-s.deleteQueue:
				if !ok {
					flush()
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
						flush()
					}
				}

			case <-ticker.C:
				flush()
			}
		}
	}()
}

func (s *URLService) Close() {
	s.closeOnce.Do(func() {
		close(s.deleteQueue)
		s.deleteWG.Wait()
	})
}
