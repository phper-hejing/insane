package server

import (
	"encoding/json"
	"fmt"
	"github.com/donnie4w/go-logger/logger"
	"github.com/tidwall/gjson"
	"insane/constant"
	"sync"
)

type ScriptRequest struct {
	Data           []gjson.Result    `json:data`
	ScriptResponse []*ScriptResponse `json:"response"`
}

type ScriptResponse struct {
	Name     string    `json:"name"`
	Response *Response `json:"response"`
}

func (scriptRequest *ScriptRequest) Run(serial uint64, scriptReportCh chan<- *ScriptReport, wg *sync.WaitGroup, stopCh <-chan int) {

	sentCh := make(chan bool)
	responseCh := make(chan *Response, 1)
	httpRequest := GenerateHttpRequest(true)

	for {
		select {
		case <-stopCh:
			logger.Debug(fmt.Sprintf("%d号事务关闭", serial))
			close(sentCh)
			close(responseCh)
			wg.Done()
			return
		default:
			scriptRequest.ScriptSend(httpRequest, sentCh, responseCh, scriptReportCh)
		}
	}
}

func (scriptRequest *ScriptRequest) ScriptSend(httpRequest *HttpRequest, sentCh chan bool, responseCh chan *Response, scriptReportCh chan<- *ScriptReport) {

	var wasteTime uint64
	resp := &Response{
		IsSuccess: false,
		ErrCode:   constant.ERROR_REQUEST_DEFAULT,
		ErrMsg:    "空数据",
	}

	defer func() {
		scriptReportCh <- &ScriptReport{
			ErrCode:        resp.ErrCode,
			ErrMsg:         resp.ErrMsg,
			ScriptResponse: scriptRequest.ScriptResponse,
			WasteTime:      wasteTime,
		}
		scriptRequest.ScriptResponse = make([]*ScriptResponse, 0)
	}()

	for _, v := range scriptRequest.Data {
		httpRequest.Parse(v.Get("data"))

		go httpRequest.HttpSend(responseCh, sentCh)
		<-sentCh
		resp = <-responseCh

		wasteTime += resp.WasteTime
		if resp.IsSuccess == false {
			return
		}

		scriptRequest.ScriptResponse = append(scriptRequest.ScriptResponse, &ScriptResponse{
			Name:     v.Get("data.name").String(),
			Response: resp,
		})
	}
}

func (scriptRequest *ScriptRequest) Validate() (vc []byte, err error) {
	sentCh := make(chan bool)
	responseCh := make(chan *Response, 1)
	scriptReportCh := make(chan *ScriptReport)
	httpRequest := GenerateHttpRequest(true)

	go scriptRequest.ScriptSend(httpRequest, sentCh, responseCh, scriptReportCh)
	resp := <-scriptReportCh

	vc, err = json.Marshal(resp)
	return
}
