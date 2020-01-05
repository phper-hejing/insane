package api

import (
	"fmt"
	"github.com/donnie4w/go-logger/logger"
	"io"
	"os"
	"strings"
)

type UploadMessage struct {
	Message
}

func (uploadMessage *UploadMessage) Do() {
	req := uploadMessage.Message.Request
	// 设置内存大小
	req.ParseMultipartForm(32 << 20)
	file, handler, err := req.FormFile("file")
	if err != nil {
		logger.Debug(err)
		return
	}
	defer file.Close()
	f, err := os.OpenFile("./upload/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)

	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	io.Copy(f, file)

	// 读取csv文件内容返回前五行
	fileNameSlice := strings.Split(handler.Filename, ".")
	suffix := fileNameSlice[len(fileNameSlice) - 1]
	if suffix == "csv" {

	}
}
