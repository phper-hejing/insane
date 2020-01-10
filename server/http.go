package server

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/donnie4w/go-logger/logger"
	"insane/constant"
	"insane/general/base/appconfig"
	"insane/utils"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

var i int64
var m sync.Mutex

const HTTP_RESPONSE_TIMEOUT = time.Duration(5) * time.Second

type HttpRequest struct {
	Url          string            `json:"url"`    // 请求域名
	Method       string            `json:"method"` // 请求方法
	Cookie       string            `json:"cookie"`
	Header       map[string]string `json:"header"`
	HttpBody     *HttpBody         `json:"body"`
	ReadResponse bool              `json:"-"`
	stop         chan int          `json:"-"`
	client       *http.Client      `json:"-"`
}

type HttpBody struct {
	Body         []*BodyField  `json:"body"`
	BodyFileData *BodyFileData `json:"-"`
}

type BodyField struct {
	Name    string      `json:"name"`
	Type    string      `json:"type"` // int|string
	Len     int64       `json:"len"`
	Default interface{} `json:"default"`
}

type BodyFileData struct {
	Index  uint64     `json:"index"`
	Column []string   `json:"column"`
	Data   [][]string `json:"data"`
}

func GenerateHttpRequest(ReadResponse bool) *HttpRequest {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConnsPerHost: appconfig.GetConfig().Http.MaxIdleConnsPerHost,
		DisableCompression:  false,
		DisableKeepAlives:   true,
	}
	return &HttpRequest{
		// Timeout: 10 * time.Second 连接超时是等待响应之后判断请求消耗时间来决定是否超时，暂时没找到解决方案
		// 临时解决：在每个协程里面增加一个定时器，如果超时，直接丢弃该协程，开启下一个协程进行Http请求
		client:       &http.Client{Transport: tr},
		ReadResponse: ReadResponse,
	}
}

func (httpRequest *HttpRequest) Http(ch chan<- *Response, wg *sync.WaitGroup) {
	sentCh := make(chan bool)
	for {
		select {
		case <-httpRequest.stop:

			m.Lock()
			i++
			logger.Debug(fmt.Sprintf("%d号协程关闭", i))
			if i == int64(cap(httpRequest.stop)) {
				i = 0
			}
			m.Unlock()

			close(sentCh)
			wg.Done()
			return
		default:
			go httpRequest.HttpSend(ch, sentCh)
			<-sentCh
		}
	}
}

func (httpRequest *HttpRequest) HttpSend(ch chan<- *Response, sentCh chan bool) {
	var (
		status    = false
		isSuccess = false
		errCode   = http.StatusInternalServerError
		errMsg    = ""
		respData  = make([]byte, 0)
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
	defer func() {
		status = true
		end := utils.Now()
		resp.ErrCode = errCode
		resp.ErrMsg = errMsg
		resp.IsSuccess = isSuccess
		resp.WasteTime = uint64(end - start)
		resp.Data = respData

		if err := recover(); err != nil {
			logger.Debug(err)
			resp.ErrMsg = err.(error).Error()
		}
		httpSendSentCh(sentCh)
		httpSendRespCh(ch, resp)
	}()

	req, err := getHttpRequest(httpRequest)
	if err != nil {
		resp.ErrCode = constant.ERROR_REQUEST_CREATED // 创建连接失败
		resp.ErrMsg = err.Error()
		return
	}

	rp, err := httpRequest.client.Do(req)
	if err != nil {
		resp.ErrCode = constant.ERROR_REQUEST_CONNECTION // 连接失败
		resp.ErrMsg = err.Error()
		return
	}

	isSuccess, errCode, respData, errMsg = httpRequest.verify(rp)
}

func (httpRequest *HttpRequest) verify(resp *http.Response) (isSuccess bool, code int, respData []byte, msg string) {
	defer resp.Body.Close()
	// 是否读取响应内容
	if httpRequest.ReadResponse {
		respData, _ = ioutil.ReadAll(resp.Body)
	}

	code = resp.StatusCode
	msg = resp.Status
	if code == http.StatusOK {
		isSuccess = true
		return
	}
	return
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

func getHttpRequest(request *HttpRequest) (req *http.Request, err error) {
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

func getBody(request *HttpRequest) io.Reader {
	var body string
	var tp string
	if request.Header != nil {
		tp = request.Header["content-type"]
	}
	switch tp {
	case "application/x-www-form-urlencoded":
		body = CreateFormBody(request.HttpBody)
	case "application/json":
		body = CreateJsonBody(request.HttpBody)
	default:
		body = CreateJsonBody(request.HttpBody)
	}
	return strings.NewReader(body)
}

func CreateJsonBody(httpBody *HttpBody) string {
	body := make(map[string]interface{})
	for _, v := range httpBody.Body {
		if v.Default == nil || v.Default == "" {
			body[v.Name] = httpBody.getValue(v)
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

func CreateFormBody(httpBody *HttpBody) string {
	body := url.Values{}
	for _, v := range httpBody.Body {
		if v.Default == nil || v.Default == "" {
			body.Set(v.Name, utils.ConvString(httpBody.getValue(v)))
		} else {
			body.Set(v.Name, utils.ConvString(v.Default))
		}
	}
	return body.Encode()
}

func (httpBody *HttpBody) getValue(bodyField *BodyField) (val interface{}) {
	switch bodyField.Type {
	case "int":
		val = utils.GetRandomintegers(bodyField.Len)
	case "string":
		val = utils.GetRandomStrings(bodyField.Len)
	case "file":
		val = httpBody.getFileValue(bodyField.Name)
	default:
		val = utils.GetRandomStrings(bodyField.Len)
	}
	return
}

func (httpBody *HttpBody) getFileValue(field string) (val interface{}) {

	if httpBody.BodyFileData == nil {
		panic(errors.New("未初始化数据文件"))
	}

	if len(httpBody.BodyFileData.Column) == 0 {
		panic(errors.New("数据文件为空"))
	}

	n := sort.Search(len(httpBody.BodyFileData.Column), func(i int) bool {
		return field == httpBody.BodyFileData.Column[i]
	})
	if n > len(httpBody.BodyFileData.Column) {
		panic(errors.New(fmt.Sprintf("%s字段不存在数据文件中", field)))
	}

	if httpBody.BodyFileData.Index == uint64(len(httpBody.BodyFileData.Data)) {
		httpBody.BodyFileData.Index = 0
	}
	httpBody.BodyFileData.Index += 1

	return httpBody.BodyFileData.Data[httpBody.BodyFileData.Index][n]
}
