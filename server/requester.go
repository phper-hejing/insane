package server

import (
	"errors"
	"github.com/donnie4w/go-logger/logger"
	"insane/utils"
	"sync"
	"time"
)

type Request struct {
	// 请求赋值
	Url         string            `json:"url"`        // 请求域名
	ConCurrency uint64            `json:"conCurrent"` // 并发数
	Duration    uint64            `json:"duration"`   // 持续时间（秒）
	Interval    int32             `json:"interval"`   // 请求间隔时间
	Method      string            `json:"method"`     // 请求方法
	Form        string            `json:"form"`       // http|websocket|tcp
	Header      map[string]string `json:"header"`
	Cookie      string            `json:"cookie"`
	Body        []*BodyField      `json:"body"`

	// 系统赋值
	Id     string `json:"id"`
	Status bool   `json:"status"`
	stop   chan int
	Report *Report `json:"report"`
}

type BodyField struct {
	Name    string      `json:"name"`
	Type    string      `json:"type"` // int|string
	Len     int64       `json:"len"`
	Default interface{} `json:"default"`
}

type Response struct {
	WasteTime uint64 // 消耗时间（毫秒）
	IsSuccess bool   // 是否请求成功
	ErrCode   int    // 错误码
	ErrMsg    string // 错误提示
}

const (
	TYPE_HTTP      = "http"
	TYPE_WEBSOCKET = "websocket"
)

func (request *Request) Dispose() {
	ch := make(chan *Response, 1000)

	var (
		wg          sync.WaitGroup // 请求完成
		wgReceiving sync.WaitGroup // 请求数据统计完成
	)

	// 统计数据
	wgReceiving.Add(1)
	go request.Report.ReceivingResults(request.Id, request.ConCurrency, ch, &wgReceiving)

	// request.duration时间后,结束所有请求
	go request.timeClosure()

	for i := uint64(0); i < request.ConCurrency; i++ {
		wg.Add(1)
		switch request.Form {
		case TYPE_HTTP:
			go Http(ch, &wg, request)
		case TYPE_WEBSOCKET:
			go Websocket(ch, &wg, request)
		default:
			wg.Done()
		}
	}

	wg.Wait()
	// 延时1毫秒 确保数据都处理完成了
	time.Sleep(1 * time.Millisecond)
	close(ch)
	close(request.stop)
	request.Status = true

	wgReceiving.Wait()
	logger.Debug("dispose out...")
}

func (request *Request) Close() (err error) {
	defer func() {
		if err2 := recover(); err2 != nil {
			err = errors.New("fail")
		}
	}()
	request.closeRequest()
	return
}

func (request *Request) VerifyParam() (err error) {
	if request.Url == "" || request.Form == "" || request.ConCurrency == 0 || request.Duration == 0 {
		err = errors.New("参数缺少")
	}
	return
}

func (request *Request) timeClosure() {
	// recover一下，避免提前结束任务后,关闭stop导致的panic
	defer func() {
		if err := recover(); err != nil {
			logger.Debug(err)
		}
	}()

	t := time.After(time.Duration(request.Duration) * time.Second)
	<-t

	if !request.Status { // 如果请求正在执行，终止它
		request.closeRequest()
	}
}

func (request *Request) initStopCh() {
	if request.ConCurrency > 0 {
		request.stop = make(chan int, request.ConCurrency)
	}
}

func (request *Request) closeRequest() {
	for i := uint64(0); i < request.ConCurrency; i++ {
		request.stop <- 1
	}
	logger.Debug("close signal len: ", len(request.stop))
}

func (bodyField *BodyField) getValue() (val interface{}) {
	switch bodyField.Type {
	case "int":
		val = utils.GetRandomintegers(bodyField.Len)
	case "string":
		val = utils.GetRandomStrings(bodyField.Len)
	default:
		val = utils.GetRandomStrings(bodyField.Len)
	}
	return
}
