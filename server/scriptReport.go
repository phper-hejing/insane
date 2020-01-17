package server

import (
	"encoding/json"
	"fmt"
	"insane/general/base/appconfig"
	"insane/utils"
	"net/http"
	"sync"
)

type ScriptReportList struct {
	ScriptReport   map[uint64][]*ScriptReport `json:"scriptReport"`
	TotalSuccess   uint64                     `json:"totalSuccess"`
	TotalError     uint64                     `json:"totalError"`
	AverageSuccess map[uint64]uint64          `json:"averageSuccess"`
	AverageError   map[uint64]uint64          `json:"averageError"`
	ErrCode        map[int]uint64             `json:"errCode"`
	ErrCodeMsg     map[int]string             `json:"errCodeMsg"`
	Status         bool                       `json:"status"`
}

type ScriptReport struct {
	ScriptResponse []*ScriptResponse `json:"scriptResponse"`
	WasteTime      uint64            `json:"wasteTime"` // 事务消耗时间
	ErrCode        int               `json:"errCode"`   // 错误码
	ErrMsg         string            `json:"errMsg"`    // 错误提示
}

const SCRIPT_REPORT_SEP = 60

func (scriptReportList *ScriptReportList) ReceivingResults(id string, conCurrency uint64, slCh <-chan *ScriptReport, wgReceiving *sync.WaitGroup) {
	defer wgReceiving.Done()

	var (
		errCode        = make(map[int]uint64)
		averageSuccess = make(map[uint64]uint64)
		averageError   = make(map[uint64]uint64)
		errCodeMsg     = make(map[int]string)
		totalSuccess   = 0
		totalError     = 0
	)

	startTime := utils.Now()
	for data := range slCh {
		curSecond := utils.CurSecond(uint64(startTime))
		// 统计维度分钟
		sep := curSecond / SCRIPT_REPORT_SEP
		if sep == 0 {
			sep = 1
		}

		if data.ErrCode == http.StatusOK {
			totalSuccess++
			averageSuccess[sep]++
		} else {
			totalError++
			averageError[sep]++
			errCode[data.ErrCode]++
			errCodeMsg[data.ErrCode] = data.ErrMsg
		}

		scriptReportList.TotalSuccess = uint64(totalSuccess)
		scriptReportList.TotalError = uint64(totalError)
		scriptReportList.AverageSuccess = averageSuccess
		scriptReportList.AverageError = averageError
		scriptReportList.ErrCode = errCode
		scriptReportList.ErrCodeMsg = errCodeMsg
		scriptReportList.ScriptReport[sep] = append(scriptReportList.ScriptReport[sep], data)
	}
	scriptReportList.Status = true

	content, err := json.Marshal(scriptReportList)
	if err == nil {
		filename := fmt.Sprintf("%s/%s.json", appconfig.GetConfig().Log.Location, id)
		utils.FileWrite(filename, string(content))
	}
}
