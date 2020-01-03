package api

import (
	"insane/server"
	"insane/utils"
)

type InfoMessage struct {
	Message
}

func (infoMessage *InfoMessage) Do() {
	report := server.TK.TaskListInfo(infoMessage.Message.InsaneRequest.Id)

	utils.Response(infoMessage.Message.ResponseWriter, utils.RspData{
		Msg:  utils.GetMsg(nil),
		Data: report,
	})
}
