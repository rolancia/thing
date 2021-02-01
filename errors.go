package server

import "errors"

var (
	ErrNoProtocol      = errors.New("ERR_NO_PROTOCOL")
	ErrInvalidProtocol = errors.New("ERR_INVALID_PROTOCOL")

	ErrConnectionClosed = errors.New("connection already closed")
)