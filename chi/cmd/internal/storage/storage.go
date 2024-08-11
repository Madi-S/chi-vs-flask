package storage

import "errors"

var (
	ErrURLNotFound = errors.New("URL Not Found")
	ErrURLAlreadyExists   = errors.New("URL Already Exists")
)
