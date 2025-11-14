package storage

import "errors"

// Storage layer errors
var (
	// ErrNotFound возвращается когда запрашиваемый ресурс не найден
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists возвращается при попытке создать ресурс который уже существует
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrConflict возвращается при конфликте данных (например, изменение merged PR)
	ErrConflict = errors.New("data conflict")
)

const (
	// UniqueViolation is a PostgreSQL error code for unique constraint violations.
	UniqueViolation = "23505"
)
