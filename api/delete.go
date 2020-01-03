package api

import (
	"github.com/donnie4w/go-logger/logger"
	"insane/server"
	"insane/utils"
)

type DeleteMessage struct {
	Message
}

func (deleteMessage *DeleteMessage) Do() {
	err := server.TK.TaskListRemove(deleteMessage.Message.InsaneRequest.Id)
	if err != nil {
		logger.Debug(err)
	}
	utils.Response(deleteMessage.Message.ResponseWriter, utils.RspData{
		Msg: utils.GetMsg(err),
	})
}
