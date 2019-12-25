package utils

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type RspData struct {
	ErrCode int         `json:"errCode"`
	Msg     string      `json:"msg"`
	Data    interface{} `json:"data"`
}

func Response(w http.ResponseWriter, data RspData) {
	b, err := json.Marshal(data)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	if err != nil {
		w.Write([]byte("error"))
	}
	w.Write(b)
}

func GetMsg(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
