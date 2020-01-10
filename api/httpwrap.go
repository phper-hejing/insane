package api

import (
	"github.com/donnie4w/go-logger/logger"
	"github.com/gorilla/websocket"
	"insane/server"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

type IMessage interface {
	Init(http.ResponseWriter, *http.Request, bool)
	Do()
}

type Message struct {
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	WsConn         *websocket.Conn
	InsaneRequest  *server.InsaneRequest
}

var upgrader = websocket.Upgrader{} // use default options

func (imsg *Message) Init(writer http.ResponseWriter, request *http.Request, isParseBody bool) {

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

	insaneReq := server.GenerateInsaneRequest()
	if isParseBody {
		// 解析请求参数
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			logger.Debug(err)
		}
		if len(body) != 0 {
			insaneReq.Parse(body)
		}
	}

	imsg.Request = request
	imsg.ResponseWriter = writer
	imsg.InsaneRequest = insaneReq
	imsg.WsConn = wsConn
}

func HandleMessage(imsg IMessage, isParseBody bool) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {

		writer.Header().Set("Access-Control-Allow-Origin", "*")
		writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		writer.Header().Set("Access-Control-Allow-Headers", "*")

		// 如果是options请求，直接返回200
		if request.Method == "OPTIONS" {
			writer.WriteHeader(200)
			return
		}

		imsg.Init(writer, request, isParseBody)
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
