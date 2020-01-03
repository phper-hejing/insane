package server

import (
	"insane/constant"
	"net/http"
	"sync"
	"time"

	"github.com/donnie4w/go-logger/logger"
	"github.com/gorilla/websocket"
)

var defaultDialer = websocket.Dialer{
	Proxy:            http.ProxyFromEnvironment,
	HandshakeTimeout: 20 * time.Second,
}

func Websocket(ch chan<- *Response, wg *sync.WaitGroup, request *Request) {

	conn, _, err := defaultDialer.Dial(request.Url, nil)

	defer func() {
		wg.Done()
	}()
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	if err != nil {
		ch <- &Response{
			WasteTime: 0,
			IsSuccess: false,
			ErrCode:   constant.ERROR_REQUEST_CONNECTION,
			ErrMsg:    err.Error(),
		}
		return
	}

	// 读取响应消息
	rstop := make(chan int, 1)
	go wsReceive(conn, ch, rstop)

	for {
		select {
		case <-request.stop:
			rstop <- 1
			return
		default:
			time.Sleep(100 * time.Millisecond)
			wsSend(conn, request)
		}
	}
}

func wsSend(conn *websocket.Conn, request *Request) (err error) {

	// 发送数据
	data := CreateJsonBody(request.Body)
	if err := conn.WriteMessage(constant.MSG_TYPE, []byte(data)); err != nil {
		logger.Debug(err)
		return err
	}
	time.Sleep(100 * time.Millisecond)
	return

}

func wsReceive(conn *websocket.Conn, ch chan<- *Response, rstop chan int) {
	for {
		select {
		case <-rstop:
			return
		default:
			// 接收数据
			_, _, err := conn.ReadMessage()
			if err != nil {
				ch <- &Response{
					WasteTime: 0,
					IsSuccess: false,
					ErrCode:   constant.ERROR_REQUEST_RECEIVE,
					ErrMsg:    "接收数据失败",
				}
			} else {
				ch <- &Response{
					WasteTime: 0,
					IsSuccess: true,
				}
			}
		}
	}
}
