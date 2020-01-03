package api

import (
	"insane/server"
)

type ServerLoadMessage struct {
	Message
}

func (serverLoadMessage *ServerLoadMessage) Do() {
	data, _ := server.InsaneLoad.Get()
	serverLoadMessage.Message.ResponseWriter.Write([]byte(data))
}
