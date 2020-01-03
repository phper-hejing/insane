package api

import (
	"github.com/donnie4w/go-logger/logger"
	"insane/server"
	"insane/utils"
)

type PushMessage struct {
	Message
}

func (pushMessage *PushMessage) Do() {
	err := server.TK.TaskListAdd(pushMessage.Message.InsaneRequest)
	if err != nil {
		logger.Debug(err)
	}
	utils.Response(pushMessage.Message.ResponseWriter, utils.RspData{
		Msg:  utils.GetMsg(err),
		Data: pushMessage.Message.InsaneRequest.Id,
	})
}
