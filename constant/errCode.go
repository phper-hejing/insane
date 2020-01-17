package constant

const (
	ERROR_REQUEST_DEFAULT    = 9999 // 通用错误
	ERROR_REQUEST_CREATED    = 5000 // 创建http.Request失败
	ERROR_REQUEST_CONNECTION = 5001 // 连接失败
	ERROR_REQUEST_RECEIVE    = 5002 // websocket接收数据失败
	ERROR_REQUEST_TIMEOUT    = 5003 // 请求超时
)
