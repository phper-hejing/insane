package constant

import "github.com/gorilla/websocket"

const (
	MSG_TYPE      = websocket.TextMessage
	MSG_HEARTBEAT = 60

	C_REGISTER = 1001
	S_REGISTER = 2001

	C_REPORT = 1002
	S_REPORT = 2002
)
