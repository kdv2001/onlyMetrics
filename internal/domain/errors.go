package domain

import "errors"

var (
	ErrNotFound         = errors.New("not found")
	ErrResourceIsLocked = errors.New("resource is locked")
)
