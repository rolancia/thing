package server

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/smallnest/goframe"
	"log"
	"net"
	"runtime/debug"
	"time"
)

type Server struct {
	context           context.Context
	frameConf         FrameConfig
	eventHandler      EventHandler
	postActionHandler PostActionHandler
}

func NewServer(ctx context.Context, frameConfig FrameConfig, eventHandler EventHandler, postActHandler PostActionHandler) *Server {
	server := Server{
		context:      ctx,
		frameConf:    frameConfig,
		eventHandler: eventHandler,
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

	// 패닉 핸들러 작동여부 확인
	ants.Submit(func() {
		panic("Panic check: defaultPool")
	})
	errGRPool.Submit(func() {
		panic("Panic check: errGRPool")
	})
	closingPool.Submit(func() {
		panic("Panic check: closingPool")
	})

	time.Sleep(3 * time.Second)
	printPanicHandlerOk()

	// 시작
	l, err := net.Listen("tcp", addr)
	if nil != err {
		return err
	}
	defer l.Close()

	log.Println("listening ", addr)
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

			newCtx, postAct := svr.eventHandler.OnJoin(uconn, &mp)
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

				err = ants.Submit(func() {
					mp := DefaultMessagePacket{}
					err := mp.Deserialize(b)
					if err != nil {
						svr.eventHandler.OnParsingFailed(uconn, b)
						return
					}

					postAct := svr.eventHandler.OnMessage(uconn, &mp)
					svr.handlePostAct(postAct, uconn)
				})
			}
		}(conn)
	}

	return nil
}

func (svr *Server) handlePostAct(act PostAction, conn *UserConn) {
	switch act {
	case PostActionNone: svr.postActionHandler.OnActionNone(act, conn)
	case PostActionClose: svr.postActionHandler.OnActionClose(act, conn)
	case PostActionBlock: svr.postActionHandler.OnActionBlock(act, conn)
	}
}

func DefaultCodec() FrameConfig {
	encoder := goframe.EncoderConfig{
		ByteOrder:                       binary.BigEndian,
		LengthFieldLength:               4,
		LengthAdjustment:                0,
		LengthIncludesLengthFieldLength: false,
	}
	decoder := goframe.DecoderConfig{
		ByteOrder:           binary.BigEndian,
		LengthFieldOffset:   0,
		LengthFieldLength:   4,
		LengthAdjustment:    0,
		InitialBytesToStrip: 4,
	}

	return FrameConfig{encoder, decoder}
}

func printPanicHandlerOk() {
	fmt.Println("-------------------------------------")
	fmt.Println("-------------------------------------")
	fmt.Println("-------------------------------------")
	fmt.Println("-------------------------------------")
	fmt.Println("-------------------------------------")
	fmt.Println("-------------------------------------")
	fmt.Println("-----SERVER PANIC HANDLERS WORK------")
}
