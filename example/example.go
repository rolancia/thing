package example

import (
	"context"
	"errors"
	"fmt"
	"github.com/rolancia/thing"
	"time"
)

func Example() {
	ctx := context.Background()

	svr := server.NewServer(ctx, server.DefaultCodec(), &exEventHandler{}, &exPostHandler{})
	fmt.Println(server.Serve(svr, "0.0.0.0", server.ErrorActionPrint))
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// event handler
type exEventHandler struct {
}

func (h *exEventHandler) OnConnected(conn *server.UserConn) (context.Context, server.PostAction) {
	conn.Conn().SetDeadline(time.Now().Add(30 * time.Second))
	conn.Async = false

	return conn.Context(), server.PostActionNone
}

// called first packet received
func (h *exEventHandler) OnJoin(conn *server.UserConn, firstMP *server.DefaultMessagePacket) (context.Context, server.PostAction) {
	// do login or ...
	authenticatedCtx := context.WithValue(conn.Context(), "id", "pizzazzang")

	return authenticatedCtx, server.PostActionNone
}

// called a packet received except first that
func (h *exEventHandler) OnMessage(conn *server.UserConn, mp *server.DefaultMessagePacket) server.PostAction {
	proto := mp.Protocol
	fmt.Println(proto)

	payload := mp.Payload

	if payload != nil {
		// do dispatch or ...
		return server.PostActionNone
	} else {
		// this will call 'OnErrorPrint'
		server.GetLandfill(conn.Context()) <- server.NewFatError(errors.New("err!"), server.ErrorActionPrint, conn)
	}

	return server.PostActionClose
}

// called before closing connection
func (h *exEventHandler) OnBeforeClose(conn *server.UserConn) {
	// cleanup
}

// called if the landfill consumes a fat error with ErrorActionPrint
func (h *exEventHandler) OnErrorPrint(serverCtx context.Context, err *server.FatError) {
	fmt.Println(err.Error())
}

// called if the landfill consumes a fat error with ErrorActionSave
func (h *exEventHandler) OnErrorSave(serverCtx context.Context, err *server.FatError) {
	// store the error into somewhere
}

// called if deserialization failed
func (h *exEventHandler) OnParsingFailed(conn *server.UserConn, data []byte) {
	// when deserialization failed. who you are?
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// post action handler
type exPostHandler struct {
}

func (h *exPostHandler) OnActionNone(act server.PostAction, conn *server.UserConn) {

}

func (h *exPostHandler) OnActionClose(act server.PostAction, conn *server.UserConn) {
	conn.Close()
}

func (h *exPostHandler) OnActionBlock(act server.PostAction, conn *server.UserConn) {
	// ban
}
