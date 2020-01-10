package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/donnie4w/go-logger/logger"
	"insane/utils"
	"io/ioutil"
	"os"
)

type DataMessage struct {
	Message
}

type dataRequest struct {
	Type     string                 `json:"type"` // get | getAll | save
	FileId   string                 `json:"fileId"`
	FileName string                 `json:"fileName"`
	FileType string                 `json:"fileType"`
	Data     map[string]interface{} `json:"data"`
}

const (
	DATA_PATH             = "./data"
	TYPE_SAVE             = "save"
	TYPE_GET              = "get"
	TYPE_GET_ALL          = "getAll"
	TYPE_DELETE           = "delete"
	FILE_TYPE_TEST_DATA   = "test_data"
	FILE_TYPE_TEST_SCRIPT = "test_script"
	FILE_TYPE_TEST_TASK   = "test_task"
)

func (dataMessage DataMessage) Do() {
	var dataReq dataRequest
	var err error
	defer func() {
		if err != nil {
			dataMessage.Message.ResponseWriter.WriteHeader(500)
		}
	}()

	body, err := ioutil.ReadAll(dataMessage.Message.Request.Body)
	if err != nil {
		logger.Debug(err)
		return
	}
	err = json.Unmarshal(body, &dataReq)
	if err != nil {
		logger.Debug(err)
		return
	}
	if dataReq.FileType == "" || dataReq.Type == "" {
		err = errors.New("type或者fileType不能为空")
		logger.Debug(err)
		return
	}

	if dataReq.Type == TYPE_GET_ALL {
		var getAllData = make([]string, 0)
		dir, err := os.OpenFile(fmt.Sprintf("%s/%s", DATA_PATH, dataReq.FileType), os.O_RDONLY, os.ModeDir)
		if err != nil {
			logger.Debug(err)
			return
		}
		defer dir.Close()
		dirInfo, _ := dir.Readdir(-1)
		for _, fileInfo := range dirInfo {
			file, err := os.Open(fmt.Sprintf("%s/%s/%s", DATA_PATH, dataReq.FileType, fileInfo.Name()))
			if err != nil {
				logger.Debug(err)
				return
			}
			dataByte, err := ioutil.ReadAll(file)
			if err != nil {
				logger.Debug(err)
				return
			}
			getAllData = append(getAllData, string(dataByte))
			file.Close()
		}
		dataJson, err := json.Marshal(getAllData)
		dataMessage.Message.ResponseWriter.Write(dataJson)
		return
	}

	if dataReq.Type == TYPE_DELETE {
		if dataReq.FileName == "" {
			err = errors.New("文件不能为空")
			logger.Debug(err)
			return
		}
		os.Remove(fmt.Sprintf("%s/%s/%s", DATA_PATH, dataReq.FileType, dataReq.FileName))
		dataMessage.Message.ResponseWriter.Write([]byte(`{"status":"success"}`))
		return
	}

	fileName := dataReq.FileName
	if fileName == "" {
		fileName = fmt.Sprintf("%d.json", utils.Now())
	}
	f, err := os.OpenFile(fmt.Sprintf("%s/%s/%s", DATA_PATH, dataReq.FileType, fileName), os.O_TRUNC|os.O_CREATE, 0666)
	defer f.Close()

	if dataReq.Type == TYPE_SAVE {

		if dataReq.FileId != "" {
			os.Remove(fmt.Sprintf("%s/%s/%s", DATA_PATH, dataReq.FileType, dataReq.FileId))
		}

		dataByte, err := json.Marshal(dataReq.Data)
		if err != nil {
			logger.Debug(err)
			return
		}
		f.Write(dataByte)
		dataMessage.Message.ResponseWriter.Write([]byte(`{"status":"success"}`))
		return
	}

	if dataReq.Type == TYPE_GET {
		dataByte, err := ioutil.ReadAll(f)
		if err != nil {
			logger.Debug(err)
			return
		}
		dataMessage.Message.ResponseWriter.Write(dataByte)
		return
	}

}
