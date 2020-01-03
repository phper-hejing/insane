package api

import (
	"encoding/json"
	"github.com/donnie4w/go-logger/logger"
	"github.com/gorilla/websocket"
	"insane/server"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

type IMessage interface {
	Init(http.ResponseWriter, *http.Request)
	Do()
}

type Message struct {
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	WsConn         *websocket.Conn
	InsaneRequest  *server.Request
}

var upgrader = websocket.Upgrader{} // use default options

func (imsg *Message) Init(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Access-Control-Allow-Origin", "*")
	writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	writer.Header().Set("Access-Control-Allow-Headers", "*")

	// 是否是websocket请求
	var wsConn *websocket.Conn
	var err error
	HeaderConnection := strings.ToLower(request.Header.Get("connection"))

	if strings.Contains(HeaderConnection, "upgrade") {
		upgrader.CheckOrigin = func(request *http.Request) bool {
			return true
		}
		wsConn, err = upgrader.Upgrade(writer, request, nil)
		if err != nil {
			logger.Debug("upgrade:", err)
			return
		}
	}

	// 解析请求参数
	insaneReq := server.GenerateRequest()
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.Debug(err)
	}
	if len(body) != 0 {
		if err := json.Unmarshal(body, insaneReq); err != nil {
			logger.Debug(err)
		}
	}

	imsg.InsaneRequest = insaneReq
	imsg.Request = request
	imsg.ResponseWriter = writer
	imsg.WsConn = wsConn
}

func HandleMessage(imsg IMessage) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		imsg.Init(writer, request)
		imsg.Do()
	}
}

var wsConnMutex sync.Mutex

func WsConnWrite(conn *websocket.Conn, msgType int, msg []byte) {
	if conn == nil {
		logger.Debug("conn is nil")
	}
	wsConnMutex.Lock()
	conn.WriteMessage(msgType, msg)
	wsConnMutex.Unlock()
}
