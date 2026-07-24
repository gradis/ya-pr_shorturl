package repository

import "errors"

var (
	ErrURLNotFound   = errors.New("URL not found")
	ErrURLIDConflict = errors.New("URL id conflict")
)
