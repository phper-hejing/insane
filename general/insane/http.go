package insane

import (
	"insane/api"
	"net/http"
	"time"

	"insane/general/base/appconfig"

	"github.com/donnie4w/go-logger/logger"
)

type InsaneHttp struct {
	http http.Server
}

var insaneHttp InsaneHttp

func OnStart() {
	RegisterRoutesHandle()
	HttpConfigInit()
	if err := insaneHttp.http.ListenAndServe(); err != nil {
		logger.Debug(err)
	}
}

func HttpConfigInit() {
	insaneHttp.http.Addr = appconfig.GetConfig().Http.Bind
	insaneHttp.http.ReadTimeout = 2 * time.Minute
	insaneHttp.http.WriteTimeout = 2 * time.Minute
	insaneHttp.http.MaxHeaderBytes = 1 << 20
}

func RegisterRoutesHandle() {
	http.HandleFunc("/request", api.Push)
	http.HandleFunc("/info", api.Info)
	http.HandleFunc("/del", api.Del)
	http.HandleFunc("/ws", api.WsInfo)
	http.HandleFunc("/serverLoad", api.ServerLoad)

	// 资源引用
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./frontend/"))))

	http.HandleFunc("/test", api.Test)
}
