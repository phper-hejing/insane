package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/donnie4w/go-logger/logger"
	"insane/general/base/appconfig"
	"insane/utils"
	"io"
	"os"
	"strings"
)

type UploadMessage struct {
	Message
}

type UploadResp struct {
	FileName string     `json:"fileName"`
	FileData [][]string `json:"fileData"`
	FileRow  int        `json:"fileRow"`
	FileCol  int        `json:"fileCol"`
}

func (uploadMessage *UploadMessage) Do() {

	fileName := ""
	fileData := make([][]string, 0)
	fileRow := 0
	fileCol := 0

	req := uploadMessage.Message.Request
	// 设置内存大小
	req.ParseMultipartForm(32 << 20)
	file, handler, err := req.FormFile("file")
	if err != nil {
		logger.Debug(err)
		return
	}
	defer file.Close()

	fileNameSlice := strings.Split(handler.Filename, ".")
	suffix := fileNameSlice[len(fileNameSlice)-1]
	fileName = fmt.Sprintf("%d.%s", utils.Now(), suffix)
	filePath := fmt.Sprintf("%s/%s", appconfig.GetConfig().File.UploadPath, fileName)
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0666)

	if err != nil {
		logger.Debug(err)
		return
	}
	defer f.Close()
	io.Copy(f, file)

	// 读取csv文件内容返回前五行
	if suffix == "csv" {
		f.Seek(0, 0) // 将文件指针移到开头
		csvData := csv.NewReader(f)
		record, err := csvData.ReadAll()
		if err != nil {
			logger.Debug(err)
			return
		}
		fileRow = len(record)
		fileCol = len(record[0])
		for _, v := range record[:5] {
			fileData = append(fileData, v)
		}
	}

	uploadResp := UploadResp{
		FileName: fileName,
		FileData: fileData,
		FileRow:  fileRow,
		FileCol:  fileCol,
	}
	resp, _ := json.Marshal(uploadResp)
	uploadMessage.Message.ResponseWriter.Write(resp)
}
