package server

import (
	"encoding/json"
	"fmt"
	"github.com/donnie4w/go-logger/logger"
	"insane/constant"
	"insane/utils"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var i int64
var m sync.Mutex

const HTTP_RESPONSE_TIMEOUT = time.Duration(5) * time.Second

func Http(ch chan<- *Response, wg *sync.WaitGroup, request *Request) {
	sentCh := make(chan bool)
	for {
		select {
		case <-request.stop:

			m.Lock()
			i++
			logger.Debug(fmt.Sprintf("%d号协程关闭", i))
			if i == int64(cap(request.stop)) {
				i = 0
			}
			m.Unlock()

			close(sentCh)
			wg.Done()
			return
		default:
			go httpSend(request.client, request, ch, sentCh)
			<-sentCh
		}
	}
}

func httpSend(client *http.Client, request *Request, ch chan<- *Response, sentCh chan bool) {
	var (
		status    = false
		isSuccess = false
		errCode   = http.StatusOK
		errMsg    = ""
		start     = utils.Now()
	)
	resp := new(Response)
	go func() {
		t := time.NewTicker(HTTP_RESPONSE_TIMEOUT)
		<-t.C
		if status == false {
			httpSendSentCh(sentCh)
		}
	}()

	req, err := getHttpRequest(request)
	if err != nil {
		resp.ErrCode = constant.ERROR_REQUEST_CREATED // 创建连接失败
		resp.ErrMsg = err.Error()
		return
	}

	rp, err := client.Do(req)
	if err != nil {
		resp.ErrCode = constant.ERROR_REQUEST_CONNECTION // 连接失败
		resp.ErrMsg = err.Error()
		return
	}

	isSuccess, errCode, errMsg = verify(rp)
	end := utils.Now()
	resp.ErrCode = errCode
	resp.ErrMsg = errMsg
	resp.IsSuccess = isSuccess
	resp.WasteTime = uint64(end - start)

	httpSendRespCh(ch, resp)
	httpSendSentCh(sentCh)
}

func httpSendSentCh(sentCh chan bool) {
	defer func() {
		if err := recover(); err != nil {
			//logger.Debug(err)
		}
	}()
	sentCh <- true
}

func httpSendRespCh(respCh chan<- *Response, response *Response) {
	defer func() {
		if err := recover(); err != nil {
			//logger.Debug(err)
		}
	}()
	respCh <- response
}

func verify(resp *http.Response) (isSuccess bool, code int, msg string) {
	defer resp.Body.Close()
	code = resp.StatusCode
	msg = resp.Status
	if code == http.StatusOK {
		isSuccess = true
		return
	}
	return
}

func getHttpRequest(request *Request) (req *http.Request, err error) {
	body := getBody(request)
	req, err = http.NewRequest(request.Method, request.Url, body)
	setHeader(request.Header, req)
	setCookie(request.Cookie, req)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func setHeader(header map[string]string, req *http.Request) {
	// default content-type:application/json
	req.Header.Add("Content-Type", "application/json")
	for k, v := range header {
		if k != "" && v != "" {
			req.Header.Add(k, v)
		}
	}
}

func setCookie(ck string, req *http.Request) {
	cookies := strings.Split(ck, "; ")
	for _, v := range cookies {
		s := strings.Split(v, "=")
		if len(s) > 1 {
			httpCk := http.Cookie{Name: s[0], Value: s[1]}
			req.AddCookie(&httpCk)
		}
	}
}

func getBody(request *Request) io.Reader {
	var body string
	var tp string
	if request.Header != nil {
		tp = request.Header["content-type"]
	}
	switch tp {
	case "application/x-www-form-urlencoded":
		body = createFormBody(request.Body)
	case "application/json":
		body = CreateJsonBody(request.Body)
	default:
		body = CreateJsonBody(request.Body)
	}
	return strings.NewReader(body)
}

func CreateJsonBody(bodyField []*BodyField) string {
	body := make(map[string]interface{})
	for _, v := range bodyField {
		if v.Default == nil || v.Default == "" {
			body[v.Name] = v.getValue()
		} else {
			body[v.Name] = v.Default
		}
	}
	s, err := json.Marshal(body)
	if err != nil {
		return ""
	}
	return string(s)
}

func createFormBody(bodyField []*BodyField) string {
	body := url.Values{}
	for _, v := range bodyField {
		if v.Default == nil {
			body.Set(v.Name, utils.ConvString(v.getValue()))
		} else {
			body.Set(v.Name, utils.ConvString(v.Default))
		}
	}
	return body.Encode()
}
