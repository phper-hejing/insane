package api

import (
	"fmt"
	"github.com/donnie4w/go-logger/logger"
)

type TestMessage struct {
	Message
}

func (testMessage *TestMessage) Do() {
	load, err := testMessage.Message.InsaneRequest.Capacity()
	if err != nil {
		logger.Debug(err)
		return
	}
	testMessage.ResponseWriter.Write([]byte(fmt.Sprintf("%.2f", load)))
}
