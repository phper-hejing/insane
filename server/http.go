package server

import (
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/donnie4w/go-logger/logger"
	"github.com/tidwall/gjson"
	"insane/constant"
	"insane/general/base/appconfig"
	"insane/utils"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const HTTP_RESPONSE_TIMEOUT = time.Duration(5) * time.Second
const HTTP_RESPONSE_FIELD_SEP = "---"

type HttpRequest struct {
	Name         string            `json:"name"`
	Url          string            `json:"url"`    // 请求域名
	Method       string            `json:"method"` // 请求方法
	Cookie       string            `json:"cookie"`
	Header       map[string]string `json:"header"`
	HttpBody     *HttpBody         `json:"body"`
	HttpResponse map[string]string `json:"-"`
	ReadResponse bool              `json:"-"`
	client       *http.Client      `json:"-"`
}

type HttpBody struct {
	Body         []*BodyField             `json:"body"`
	BodyFileData map[string]*BodyFileData `json:"-"`
}

type BodyField struct {
	Name    string      `json:"name"`
	Type    string      `json:"type"` // int|string
	Len     int64       `json:"len"`
	Default interface{} `json:"default"`
	Dynamic string      `json:"dynamic"`
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
		client: &http.Client{Transport: tr},
		HttpBody: &HttpBody{
			Body:         make([]*BodyField, 0),
			BodyFileData: make(map[string]*BodyFileData),
		},
		HttpResponse: make(map[string]string),
		ReadResponse: ReadResponse,
	}
}

func (httpRequest *HttpRequest) Parse(data gjson.Result) {
	httpRequest.Name = data.Get("name").String()
	httpRequest.Url = data.Get("url").String()
	httpRequest.Method = data.Get("method").String()
	httpRequest.Cookie = data.Get("cookie").String()
	json.Unmarshal([]byte(data.Get("header").String()), &httpRequest.Header)
	json.Unmarshal([]byte(data.Get("body").String()), &httpRequest.HttpBody.Body)
}

func (httpRequest *HttpRequest) Run(serial uint64, ch chan<- *Response, wg *sync.WaitGroup, stopCh <-chan int) {
	sentCh := make(chan bool)
	for {
		select {
		case <-stopCh:
			logger.Debug(fmt.Sprintf("%d号协程关闭", serial))
			close(sentCh)
			wg.Done()
			return
		default:
			go httpRequest.HttpSend(ch, sentCh)
			<-sentCh
		}
	}
}

func (httpRequest *HttpRequest) HttpSend(respCh chan<- *Response, sentCh chan bool) {
	var (
		status    = false
		isSuccess = false
		errCode   = http.StatusOK
		errMsg    = ""
		respData  = make([]byte, 0)
		start     = utils.Now()
	)
	resp := new(Response)
	go func() {
		t := time.NewTicker(HTTP_RESPONSE_TIMEOUT)
		<-t.C
		if status == false {
			end := utils.Now()
			resp.ErrCode = constant.ERROR_REQUEST_TIMEOUT
			resp.ErrMsg = fmt.Sprintf("默认超时时间%ds, 请重试", HTTP_RESPONSE_TIMEOUT/1000000000)
			resp.IsSuccess = isSuccess
			resp.WasteTime = uint64(end - start)
			resp.Data = respData

			if err := recover(); err != nil {
				logger.Debug(err)
				resp.ErrMsg = err.(error).Error()
			}
			httpSendSentCh(sentCh)
			httpSendRespCh(respCh, resp)
		}
	}()
	defer func() {
		status = true
		end := utils.Now()
		resp.ErrCode = errCode
		resp.ErrMsg = errMsg
		resp.IsSuccess = isSuccess
		resp.WasteTime = uint64(end - start)
		resp.Data = string(respData)

		if err := recover(); err != nil {
			logger.Debug(err)
			resp.ErrMsg = err.(error).Error()
		}
		httpSendSentCh(sentCh)
		httpSendRespCh(respCh, resp)
	}()

	req, err := httpRequest.getRequest()
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
		key := httpRequest.Name
		if key == "" {
			key = httpRequest.Url
		}
		httpRequest.HttpResponse[key] = string(respData)
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
			logger.Debug(err)
		}
	}()
	sentCh <- true
}

func httpSendRespCh(respCh chan<- *Response, response *Response) {
	defer func() {
		if err := recover(); err != nil {
			logger.Debug(err)
		}
	}()
	respCh <- response
}

func (request *HttpRequest) getRequest() (req *http.Request, err error) {
	body := request.getBody()
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

func (request *HttpRequest) getBody() io.Reader {
	var body string
	switch request.Header["content-type"] {
	case "application/x-www-form-urlencoded":
		body = request.createFormBody()
	case "application/json":
		body = request.createJsonBody()
	default:
		body = request.createJsonBody()
	}
	logger.Info("http send body: ", body)
	return strings.NewReader(body)
}

func (request *HttpRequest) createJsonBody() string {
	body := make(map[string]interface{})
	for _, v := range request.HttpBody.Body {
		if v.Default == nil || v.Default == "" {
			body[v.Name] = request.getBodyValue(v)
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

func (request *HttpRequest) createFormBody() string {
	body := url.Values{}
	for _, v := range request.HttpBody.Body {
		if v.Default == nil || v.Default == "" {
			body.Set(v.Name, utils.ConvString(request.getBodyValue(v)))
		} else {
			body.Set(v.Name, utils.ConvString(v.Default))
		}
	}
	return body.Encode()
}

func (request *HttpRequest) getBodyValue(bodyField *BodyField) (val interface{}) {
	switch bodyField.Type {
	case "int":
		val = utils.GetRandomintegers(bodyField.Len)
	case "string":
		val = utils.GetRandomStrings(bodyField.Len)
	case "file":
		val = request.getFileValue(bodyField.Dynamic)
	case "response":
		val = request.getResponseValue(bodyField.Dynamic)
	default:
		val = utils.GetRandomStrings(bodyField.Len)
	}
	return
}

func (request *HttpRequest) getFileValue(fileInfo string) (val interface{}) {

	info := strings.Split(fileInfo, "---")
	if len(info) != 2 {
		panic(request.getErrorMsg("文件信息获取失败"))
	}

	fileName := info[0]
	field := info[1]

	if fileName == "" {
		panic(request.getErrorMsg("文件名不能为空"))
	}

	fileData, ok := request.HttpBody.BodyFileData[fileName]
	if !ok {
		file, err := os.Open(fmt.Sprintf("./%s/%s", appconfig.GetConfig().File.UploadPath, fileName))
		if err != nil {
			panic(request.getErrorMsg(err.Error()))
		}
		csv := csv.NewReader(file)
		csvData, err := csv.ReadAll()
		if err != nil {
			panic(request.getErrorMsg(err.Error()))
		}
		fileData = &BodyFileData{
			Index:  0,
			Column: csvData[0],
			Data:   csvData[1:],
		}
	}

	n := -1
	for k, v := range fileData.Column {
		if v == field {
			n = k
		}
	}
	if n == -1 {
		panic(request.getErrorMsg(fmt.Sprintf("%s字段不存在数据文件中", field)))
	}

	if fileData.Index == uint64(len(fileData.Data)) {
		fileData.Index = 0
	}
	fileData.Index += 1

	request.HttpBody.BodyFileData[fileName] = fileData
	return fileData.Data[fileData.Index][n]
}

func (request *HttpRequest) getResponseValue(field string) (val interface{}) {
	fieldArr := strings.Split(field, HTTP_RESPONSE_FIELD_SEP)
	respData, ok := request.HttpResponse[fieldArr[0]]
	if len(fieldArr) <= 1 || !ok {
		panic(request.getErrorMsg("解析Response字段失败"))
	}

	jsonStr := ""
	for _, v := range fieldArr[1:] {
		jsonStr += v + "."
	}
	jsonStr = jsonStr[:len(jsonStr)-1]

	// 解析json数据
	value := gjson.Get(respData, jsonStr)
	if !value.Exists() {
		panic(request.getErrorMsg(fmt.Sprintf("%s没有%s字段", request.Name, jsonStr)))
	}
	return value.String()
}

func (request *HttpRequest) getErrorMsg(msg string) error {
	return errors.New(fmt.Sprintf("URL：%s   错误描述：%s", msg, request.Url))
}
