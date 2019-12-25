package api

import (
	"encoding/json"
	"fmt"
	"github.com/donnie4w/go-logger/logger"
	"github.com/gorilla/websocket"
	"insane/server"
	"insane/utils"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

func Push(w http.ResponseWriter, request *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	var req server.Request
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.Debug(err)
	}
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Debug(err)
	}

	err = server.TK.TaskListAdd(&req)

	utils.Response(w, utils.RspData{
		Msg:  utils.GetMsg(err),
		Data: req.Id,
	})
}

func Del(w http.ResponseWriter, request *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	var req server.Request
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.Debug(err)
	}
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Debug(err)
	}

	err = server.TK.TaskListRemove(req.Id)

	utils.Response(w, utils.RspData{
		Msg: utils.GetMsg(err),
	})
}

func Info(w http.ResponseWriter, request *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	var req server.Request
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.Debug(err)
	}
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Debug(err)
	}

	report := server.TK.TaskListInfo(req.Id)

	utils.Response(w, utils.RspData{
		Msg:  utils.GetMsg(err),
		Data: report,
	})
}

func ServerLoad(w http.ResponseWriter, req *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	data, _ := server.InsaneLoad.Get()
	w.Write([]byte(data))
}

func Test(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("test"))
}

var upgrader = websocket.Upgrader{} // use default options
var write = func(conn *websocket.Conn, m *sync.Mutex, msgType int, msg []byte) {
	m.Lock()
	conn.WriteMessage(msgType, msg)
	m.Unlock()
}

func WsInfo(w http.ResponseWriter, r *http.Request) {

	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	var m = new(sync.Mutex)
	// 心跳
	go func() {
		t := time.NewTicker(5 * time.Second)
		for {
			if c == nil {
				logger.Debug("ws close")
				return
			}
			<-t.C
			write(c, m, websocket.TextMessage, []byte(`{"type":"ping"}`))
		}
	}()

	id := ""
	for {
		mt, message, err := c.ReadMessage()

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
							write(c, m, mt, []byte(fmt.Sprintf(`{"type":"rspReport","data":%s}`, report)))
							return
						}
						report := server.TK.TaskListInfo(id)
						write(c, m, mt, []byte(fmt.Sprintf(`{"type":"rspReport","data":%s}`, report)))
					}
				}()
			}
		}

	}
}
