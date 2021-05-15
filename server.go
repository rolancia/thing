package server

import (
	"context"
	"encoding/binary"
	"github.com/panjf2000/ants/v2"
	"github.com/smallnest/goframe"
	"log"
	"net"
	"runtime/debug"
)

type Server struct {
	context           context.Context
	frameConf         FrameConfig
	eventHandler      EventHandler
	postActionHandler PostActionHandler
}

func NewServer(ctx context.Context, frameConfig FrameConfig, eventHandler EventHandler, postActHandler PostActionHandler) *Server {
	server := Server{
		context:           ctx,
		frameConf:         frameConfig,
		eventHandler:      eventHandler,
		postActionHandler: postActHandler,
	}

	return &server
}

func defaultPanicHandler(v interface{}) {
	log.Println(v)
	debug.PrintStack()
}

func Serve(svr *Server, addr string, internalMode ErrorAction) error {
	// 에러 핸들러
	svr.context = WithLandfill(svr.context)
	chErr := GetLandfill(svr.context)
	errGRPool, err := ants.NewPool(100)
	if err != nil {
		return err
	}

	go func() {
		for {
			err := <-chErr
			switch err.Action {
			case ErrorActionPrint:
				err := errGRPool.Submit(func() {
					svr.eventHandler.OnErrorPrint(svr.context, err)
				})
				if err != nil {
					return
				}
			case ErrorActionSave:
				err := errGRPool.Submit(func() {
					svr.eventHandler.OnErrorSave(svr.context, err)
				})
				if err != nil {
					return
				}
			}
		}
	}()

	// 클로징 핸들러
	// 이 채널을 통해 클로즈해야 핸들러의 OnBeforeClose가 호출됨
	chConnClosed := make(chan *UserConn, 10000)
	closingPool, err := ants.NewPool(100)
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			if v := recover(); v != nil {
				defaultPanicHandler(v)
			}
		}()
		for {
			closingConn := <-chConnClosed

			err := closingPool.Submit(func() {
				svr.eventHandler.OnBeforeClose(closingConn)
				closingConn.Close()
			})
			if err != nil {
				return
			}
		}
	}()

	// 시작
	l, err := net.Listen("tcp", addr)
	if nil != err {
		return err
	}
	defer l.Close()

	log.Println("listening", addr)
	for {
		conn, err := l.Accept()
		if nil != err {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				chErr <- NewFatError(ne, internalMode, nil)
				continue
			}

			panic(err)
		}

		go func(conn net.Conn) {
			uconn := newUserConn(svr.context, &svr.frameConf, conn, chConnClosed)
			defer func() {
				if v := recover(); v != nil {
					defaultPanicHandler(v)
					uconn.Close()
				}
			}()

			newCtx, postAct := svr.eventHandler.OnConnected(uconn)
			uconn.context = newCtx
			svr.handlePostAct(postAct, uconn)
			if uconn.closed {
				return
			}

			b, err := uconn.ReadFrame()
			if err != nil {
				uconn.Close()
				return
			}

			mp := DefaultMessagePacket{}
			err = mp.Deserialize(b)
			if err != nil {
				svr.eventHandler.OnParsingFailed(uconn, b)
				return
			}

			newCtx, postAct = svr.eventHandler.OnJoin(uconn, &mp)
			uconn.context = newCtx

			svr.handlePostAct(postAct, uconn)
			if uconn.closed {
				return
			}

			for {
				b, err := uconn.ReadFrame()
				if err != nil {
					uconn.Close()
					return
				}

				if uconn.Async {
					_ = ants.Submit(func() {
						svr.handleMessage(uconn, b)
					})
				} else {
					svr.handleMessage(uconn, b)
				}
			}
		}(conn)
	}

	return nil
}

func (svr *Server) handleMessage(uconn *UserConn, b []byte) {
	mp := DefaultMessagePacket{}
	err := mp.Deserialize(b)
	if err != nil {
		svr.eventHandler.OnParsingFailed(uconn, b)
		return
	}

	postAct := svr.eventHandler.OnMessage(uconn, &mp)
	svr.handlePostAct(postAct, uconn)
}

func (svr *Server) handlePostAct(act PostAction, conn *UserConn) {
	svr.postActionHandler.OnPostAction(act, conn)
}

func DefaultCodec() FrameConfig {
	encoder := goframe.EncoderConfig{
		ByteOrder:                       binary.LittleEndian,
		LengthFieldLength:               4,
		LengthAdjustment:                0,
		LengthIncludesLengthFieldLength: false,
	}
	decoder := goframe.DecoderConfig{
		ByteOrder:           binary.LittleEndian,
		LengthFieldOffset:   0,
		LengthFieldLength:   4,
		LengthAdjustment:    0,
		InitialBytesToStrip: 4,
	}

	return FrameConfig{encoder, decoder}
}
