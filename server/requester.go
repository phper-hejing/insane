package server

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/donnie4w/go-logger/logger"
	"insane/general/base/appconfig"
	"insane/utils"
	"net/http"
	"strconv"
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
	Form        string            `json:"form"`       // http|websocket
	Type        string            `json:"type"`       // 请求模式 （common | capacity） default：common
	Header      map[string]string `json:"header"`
	Cookie      string            `json:"cookie"`
	Body        []*BodyField      `json:"body"`

	// 系统赋值
	Id     string `json:"id"`
	Status bool   `json:"status"`
	stop   chan int
	client *http.Client `json:"-"`
	Report *Report      `json:"report"`
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
	ADVAMCE_CPU    = 5   // 预请求前，计算cpu时间（秒）
	ADVANCE_COUNT  = 100 // 预请求协程数
	ADVANCE_DATE   = 5   // 预请求时间（秒）
)

func GenerateRequest() *Request {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConnsPerHost: appconfig.GetConfig().Http.MaxIdleConnsPerHost,
		DisableCompression:  false,
		DisableKeepAlives:   true,
	}
	return &Request{
		// Timeout: 10 * time.Second
		client: &http.Client{Transport: tr},
	}
}

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
	if request.Url == "" || request.Form == "" {
		err = errors.New("参数缺少")
	}
	return
}

func (request *Request) VerifyUrl() (err error) {
	req, err := http.NewRequest(request.Method, request.Url, nil)
	if err != nil {
		return
	}
	resp, err := request.client.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("请求错误，状态码：%d", resp.StatusCode))
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

// 智能模式
// 生成一些请求来计算服务器负载，在计算出并发数
// （生成请求数） / （cpu消耗百分比）= （单个请求消耗cpu百分比）
func (request *Request) Capacity() (avgLoad float64, err error) {

	logger.Debug("统计请求前的cpu负载...")
	// 获取最近五次的cpu百分比
	cpuLoad := InsaneLoad.GetLatelyCpuLoad(ADVAMCE_CPU)

	logger.Debug("智能模式开始...")

	if err = request.VerifyUrl(); err != nil {
		return 0, err
	}

	if request.Form == "http" {
		curlCh := make(chan uint32)
		go request.advanceHttp(curlCh)
		reqCpuLoad := request.requestCpuLoad(curlCh)

		logger.Debug("统计负载结束...")

		// 放弃负载slice的最后两个,请求结束时负载会瞬间降下来，这时统计负载的协程可能还没结束，导致统计不准确
		if len(reqCpuLoad) > 2 {
			reqCpuLoad = reqCpuLoad[0 : len(reqCpuLoad)-2]
		}
		var cpuLoadBeforeCount uint32 = 0
		for _, load := range cpuLoad {
			cpuLoadBeforeCount += load
		}
		var cpuLoadAfterCount uint32 = 0
		for _, load := range reqCpuLoad {
			cpuLoadAfterCount += load
		}

		logger.Debug(cpuLoad)
		logger.Debug(reqCpuLoad)

		reqLen := float64(len(reqCpuLoad))
		reqBeforeAvg, err := strconv.ParseFloat(fmt.Sprintf("%.2f", float64(cpuLoadBeforeCount)/ADVAMCE_CPU), 64)
		reqAfterAvg, err := strconv.ParseFloat(fmt.Sprintf("%.2f", float64(cpuLoadAfterCount)/reqLen), 64)

		// 请求中的cpu负载平均值小于请求前的cpu负载平均值
		// 可能是目标网站带宽性能等问题，导致负载上不去
		if reqBeforeAvg >= reqAfterAvg {
			return ADVANCE_COUNT, nil
		}

		reqAvg := reqAfterAvg - reqBeforeAvg
		avgLoad, err = strconv.ParseFloat(fmt.Sprintf("%.1f", reqAvg/ADVANCE_COUNT), 64)
		return avgLoad, err
	}

	if request.Form == "websocket" {

	}

	return 0, errors.New("智能模式必须是http | websocket")
}

func (request *Request) requestCpuLoad(curlCh chan uint32) (reqCpuLoad []uint32) {
	logger.Debug("统计负载开始...")
	for {
		select {
		case <-curlCh:
			return
		default:
			reqCpuLoad = append(reqCpuLoad, InsaneLoad.getCpuLoad())
		}
	}
}

func (request *Request) advanceHttp(curlCh chan uint32) {

	logger.Debug("正在执行预请求...")

	var adMutex sync.Mutex
	var reqNum uint64
	for i := 0; i < ADVANCE_COUNT; i++ {
		go func() {
			t := time.NewTicker(time.Duration(ADVANCE_DATE) * time.Second)
			for {
				select {
				case <-t.C:
					adMutex.Lock()
					reqNum++
					if reqNum >= ADVANCE_COUNT {
						curlCh <- 1
						logger.Debug("预请求结束...")
					}
					adMutex.Unlock()
					return
				default:
					req, _ := http.NewRequest(request.Method, request.Url, nil)
					resp, err := request.client.Do(req)
					if err != nil {
						logger.Debug(err)
						continue
					}
					resp.Body.Close()
				}
			}
		}()
	}
}

func (request *Request) advanceWebsocket() {

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
