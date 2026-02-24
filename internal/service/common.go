package service

import (
	"blog-api/pkg/logging"
	"errors"
)

var logger = logging.L()

var (
	ErrForbidden = errors.New("forbidden")
	ErrDatabase  = errors.New("database error")
)
