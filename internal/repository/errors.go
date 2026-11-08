package repository

import "errors"

var (
	// ErrURLNotFound указывает, что короткий ID отсутствует в хранилище.
	ErrURLNotFound = errors.New("URL not found")
	// ErrURLIDConflict указывает, что короткий ID уже занят другой записью.
	ErrURLIDConflict = errors.New("URL id conflict")
	// ErrURLDeleted указывает, что ссылка помечена как удалённая.
	ErrURLDeleted = errors.New("URL deleted")
)
