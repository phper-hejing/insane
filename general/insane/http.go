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
	http.HandleFunc("/request", api.HandleMessage(new(api.PushMessage)))
	http.HandleFunc("/info", api.HandleMessage(new(api.InfoMessage)))
	http.HandleFunc("/del", api.HandleMessage(new(api.DeleteMessage)))
	http.HandleFunc("/ws", api.HandleMessage(new(api.ReportMessage)))
	http.HandleFunc("/serverLoad", api.HandleMessage(new(api.ServerLoadMessage)))
	http.HandleFunc("/upload", api.HandleMessage(new(api.UploadMessage)))
	http.HandleFunc("/test", api.HandleMessage(new(api.TestMessage)))

	// 资源引用
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./frontend/"))))

}
