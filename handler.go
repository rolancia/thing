package server

import (
	"context"
)

type PostAction uint8

const (
	PostActionNone PostAction = iota
)

type ErrorAction uint8

const (
	ErrorActionNone ErrorAction = iota
	ErrorActionPrint
	ErrorActionSave
)

type FatError struct {
	error
	Action ErrorAction
	Conn   *UserConn
}

func NewFatError(err error, act ErrorAction, conn *UserConn) *FatError {
	return &FatError{
		error:  err,
		Action: act,
		Conn:   conn,
	}
}

type EventHandler interface {
	// context를 리턴하면 UserConn의 context가 대체됨
	OnConnected(conn *UserConn) (context.Context, PostAction)
	OnJoin(conn *UserConn, firstMP *DefaultMessagePacket) (context.Context, PostAction)
	OnMessage(conn *UserConn, mp *DefaultMessagePacket) PostAction
	OnBeforeClose(conn *UserConn)

	OnErrorPrint(serverCtx context.Context, err *FatError)
	OnErrorSave(serverCtx context.Context, err *FatError)
	OnParsingFailed(conn *UserConn, data []byte)
}

type PostActionHandler interface {
	OnPostAction(act PostAction, conn *UserConn)
}
