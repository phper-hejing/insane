package api

import (
	"encoding/json"
	"github.com/donnie4w/go-logger/logger"
	"insane/constant"
	"insane/server"
)

type ClusterMessage struct {
	Message
}

func (clusterMessage *ClusterMessage) Do() {
	var wsConn = clusterMessage.Message.WsConn
	defer func() {
		if wsConn != nil {
			wsConn.Close()
		}
	}()

	for {
		var msg server.ProtoSentMsg
		var sentData server.SentData
		_, message, err := wsConn.ReadMessage()
		if err != nil {
			logger.Debug(err)
			return
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			logger.Debug(err)
			continue
		}

		if err := json.Unmarshal(message, &sentData); err != nil {
			logger.Debug(err)
			continue
		}

		switch msg.ProtoId {
		case constant.C_REGISTER:
			clusterMessage.s_register(&sentData)
		case constant.C_REPORT:
			clusterMessage.s_report(&sentData)
		}
	}
}

func (clusterMessage *ClusterMessage) s_register(sentData *server.SentData) {
	// 添加子节点到集群列表
	var cluster server.Cluster
	clusterId := server.InsaneMaster.GenerateClusterId()
	cluster.ClusterId = clusterId
	cluster.ClusterInfo.ServerInfo = sentData.ServerInfo
	server.InsaneMaster.AddCluster(&cluster)

	protoMsg := server.ProtoReplyMsg{
		ProtoId: constant.S_REGISTER,
		ReplyData: server.ReplyData{
			ClusterId: clusterId,
		},
	}
	protoByte, err := json.Marshal(protoMsg)
	if err != nil {
		logger.Debug(err)
		return
	}
	WsConnWrite(clusterMessage.Message.WsConn, constant.MSG_TYPE, protoByte)
}

func (clusterMessage *ClusterMessage) s_report(sentData *server.SentData) {

}
