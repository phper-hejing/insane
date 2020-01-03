package api

import (
	"encoding/json"
	"fmt"
	"github.com/donnie4w/go-logger/logger"
	"insane/constant"
	"insane/server"
	"time"
)

type ReportMessage struct {
	Message
}

func (reportMessage *ReportMessage) Do() {

	var wsConn = reportMessage.Message.WsConn
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
			WsConnWrite(wsConn, constant.MSG_TYPE, []byte(`{"type":"ping"}`))
		}
	}()

	id := ""
	for {
		mt, message, err := wsConn.ReadMessage()

		if err != nil {
			logger.Debug(err)
			if id != "" {
				err = server.TK.TaskListRemove(id)
				if err != nil {
					logger.Debug(err)
				}
			}
			break
		}

		data := make(map[string]string)
		if err := json.Unmarshal(message, &data); err != nil {
			logger.Debug(err)
			continue
		}
		tp, ok := data["type"]
		if !ok {
			continue
		}

		switch tp {
		case "reqReport":
			// 这里不能直接 id, ok := data["data"]
			// 会导致新建一个id变量，而不是上面声明的id
			ok = false
			id, ok = data["data"]
			if ok {
				go func() {
					t := time.NewTicker(1 * time.Second)
					for {
						<-t.C
						if server.TK.TaskListStatus(id) == server.COMPLETED_TASK {
							// 如果任务状态已完成，最后发送一次report然后结束
							report := server.TK.TaskListInfo(id)
							WsConnWrite(wsConn, mt, []byte(fmt.Sprintf(`{"type":"rspReport","data":%s}`, report)))
							return
						}
						report := server.TK.TaskListInfo(id)
						WsConnWrite(wsConn, mt, []byte(fmt.Sprintf(`{"type":"rspReport","data":%s}`, report)))
					}
				}()
			}
		}

	}
}
