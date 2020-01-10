package api

import (
	"encoding/json"
	"errors"
	"github.com/donnie4w/go-logger/logger"
	"github.com/tidwall/gjson"
	"insane/constant"
	"insane/server"
	"insane/utils"
	"sync"
	"time"
)

type WsMessage struct {
	Message
	MessageData gjson.Result
	M           sync.Mutex
}

type WsResponse struct {
	Type  string      `json:"type"`
	Error string      `json:"error"`
	Data  interface{} `json:"data"`
}

// 任务Id，ws断开连接时，任务需要中断
var taskId = ""

const (
	WS_TYPE_PING   = "ping"
	WS_TYPE_REPORT = "report"
	WS_TYPE_SCRIPT = "test_script"
)

func (wsMessage *WsMessage) Do() {

	var wsConn = wsMessage.Message.WsConn
	defer func() {
		if wsConn != nil {
			wsConn.Close()
		}
	}()

	// 心跳
	go func() {
		t := time.NewTicker(constant.MSG_HEARTBEAT * time.Second)
		for {
			<-t.C
			if wsConn == nil {
				logger.Debug("ws close")
				return
			}
			wsMessage.send(WS_TYPE_PING, "", "")
		}
	}()

	for {
		_, message, err := wsConn.ReadMessage()

		if err != nil {
			logger.Debug(err)
			wsMessage.close()
			break
		}

		data, err := utils.ParseJson(string(message))
		if err != nil {
			logger.Debug(err)
			continue
		}
		wsType := data.Get("type").String()
		wsMessage.MessageData = data

		switch wsType {
		case WS_TYPE_REPORT:
			go wsMessage.reqReport()
		case WS_TYPE_SCRIPT:
			go wsMessage.testScript()
		}

	}
}

func (wsMessage *WsMessage) send(wsType string, wsErr string, data interface{}) {
	wsMessage.M.Lock()
	defer wsMessage.M.Unlock()
	dataByte, err := json.Marshal(WsResponse{
		Type:  wsType,
		Data:  data,
		Error: wsErr,
	})
	if err != nil {
		logger.Debug(err)
		return
	}
	wsMessage.Message.WsConn.WriteMessage(constant.MSG_TYPE, dataByte)
}

func (wsMessage *WsMessage) close() {
	if taskId != "" {
		err := server.TK.TaskListRemove(taskId)
		if err != nil {
			logger.Debug(err)
		}
	}
}

func (wsMessage *WsMessage) reqReport() {
	taskId = wsMessage.MessageData.Get("data").String()
	t := time.NewTicker(1 * time.Second)
	for {
		<-t.C
		if server.TK.TaskListStatus(taskId) == server.COMPLETED_TASK {
			// 如果任务状态已完成，最后发送一次report然后结束
			report := server.TK.TaskListInfo(taskId)
			wsMessage.send(WS_TYPE_REPORT, "", report)
			return
		}
		report := server.TK.TaskListInfo(taskId)
		wsMessage.send(WS_TYPE_REPORT, "", report)
	}
}

func (wsMessage *WsMessage) testScript() {
	err := errors.New("")
	data := wsMessage.MessageData.Get("data.data").Array()
	scriptProto := wsMessage.MessageData.Get("data.protocol").String()

	defer func() {
		wsMessage.send(WS_TYPE_SCRIPT, err.Error(), nil)
	}()

	if data == nil {
		err = errors.New("data不是一个数组")
		return
	}
	script := server.GenerateScript()
	script.Proto = scriptProto
	script.Data = data
	script.Validate()
}
