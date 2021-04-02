package server

import (
	"context"
	"github.com/smallnest/goframe"
	"net"
	"sync"
)

type UserConn struct {
	goframe.FrameConn
	Async bool

	context context.Context

	chClosed chan *UserConn
	lock     sync.RWMutex
	closed   bool
}

func (uc *UserConn) Context() context.Context {
	return uc.context
}

func (uc *UserConn) Close() error {
	if uc != nil {
		uc.lock.Lock()
		defer uc.lock.Unlock()

		if uc.closed {
			return ErrConnectionClosed
		}

		uc.chClosed <- uc
		uc.closed = true
	} else {
		return ErrConnectionClosed
	}

	return nil
}

func (uc *UserConn) WriteMP(mp *DefaultMessagePacket) error {
	uc.lock.Lock()
	defer func() {
		uc.lock.Unlock()
	}()

	if uc.closed {
		return ErrConnectionClosed
	}

	return uc.WriteFrame(mp.Serialize())
}

func newUserConn(ctx context.Context, conf *FrameConfig, conn net.Conn, chClosed chan *UserConn) *UserConn {
	fc := goframe.NewLengthFieldBasedFrameConn(conf.Enc, conf.Dec, conn)
	return &UserConn{context: ctx, FrameConn: fc, chClosed: chClosed}
}

type FrameConfig struct {
	Enc goframe.EncoderConfig
	Dec goframe.DecoderConfig
}
