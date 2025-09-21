package domain

import "errors"

var (
	// ErrNotFound ошибка сущность не найдена
	ErrNotFound = errors.New("not found")
	// ErrResourceIsLocked ошибка попытки параллельного доступа к ресурсу
	ErrResourceIsLocked = errors.New("resource is locked")
)
