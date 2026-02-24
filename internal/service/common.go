package service

import (
	"errors"

	"blog-api/pkg/logging"
)

var logger = logging.L()

var (
	ErrForbidden = errors.New("forbidden")
	ErrDatabase  = errors.New("database error")
)
