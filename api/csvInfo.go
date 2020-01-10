package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/donnie4w/go-logger/logger"
	"os"
	"strings"
)

type CsvInfoMessage struct {
	Message
}

func (csvInfoMessage CsvInfoMessage) Do() {
	pathArr := strings.Split(csvInfoMessage.Request.RequestURI, "/")
	fileName := pathArr[len(pathArr)-1]
	filePath := fmt.Sprintf("%s/%s", UPLOAD_PATH, fileName)
	f, err := os.Open(filePath)
	csv := csv.NewReader(f)
	record, err := csv.ReadAll()
	if err != nil {
		logger.Debug(err)
		return
	}
	fileData := make([][]string, 0)
	for _, v := range record[:5] {
		fileData = append(fileData, v)
	}

	uploadResp := UploadResp{
		FileName: fileName,
		FileData: fileData,
	}
	resp, _ := json.Marshal(uploadResp)
	csvInfoMessage.Message.ResponseWriter.Write(resp)
}
