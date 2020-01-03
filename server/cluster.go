package server

import (
	"encoding/json"
	"github.com/donnie4w/go-logger/logger"
	"github.com/gorilla/websocket"
	"insane/constant"
	"insane/general/base/appconfig"
)

type Cluster struct {
	ClusterId   uint64       `json:"clusterId"`
	ClusterInfo *ClusterInfo `json:"clusterInfo"`
}

type ClusterInfo struct {
	Report     Report     `json:"report"`
	ServerInfo ServerInfo `json:"serverInfo"`
}

type SentData struct {
	Report     Report     `json:"report"`
	ServerInfo ServerInfo `json:"serverInfo"`
}

type ProtoSentMsg struct {
	ProtoId  uint64   `json:"protoId"`
	SentData SentData `json:"sentData"`
}

var InsaneCluster Cluster

func (cluster *Cluster) Init() {
	cluster.ClusterInfo = new(ClusterInfo)
	cluster.ClusterInfo.ServerInfo = InsaneLoad.ServerInfo
}

func (cluster *Cluster) Register() error {
	cluster.Init()
	masterUrl := appconfig.GetConfig().Cluster.MasterUrl
	if masterUrl != "" {
		wsConn, _, err := websocket.DefaultDialer.Dial(masterUrl, nil)
		if err != nil {
			logger.Debug(err)
			return err
		}
		protoMsg := ProtoSentMsg{
			ProtoId: constant.C_REGISTER,
			SentData: SentData{
				ServerInfo: cluster.ClusterInfo.ServerInfo,
			},
		}
		protoByte, err := json.Marshal(protoMsg)
		if err != nil {
			logger.Debug(err)
			return err
		}
		wsConn.WriteMessage(constant.MSG_TYPE, protoByte)

		go func() {
			for {
				var msg ProtoReplyMsg
				var replyData ReplyData
				_, message, err := wsConn.ReadMessage()
				if err != nil {
					logger.Debug(err)
					return
				}
				if err := json.Unmarshal(message, &msg); err != nil {
					logger.Debug(err)
					continue
				}
				if err := json.Unmarshal(message, &replyData); err != nil {
					logger.Debug(err)
					continue
				}
				switch msg.ProtoId {
				case constant.S_REGISTER:
					cluster.c_register(&replyData)
				case constant.S_REPORT:
				}
			}
		}()
	}
	return nil
}

func (cluster *Cluster) c_register(replyData *ReplyData) {
	InsaneCluster.ClusterId = replyData.ClusterId
}
