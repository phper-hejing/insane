package server

import "insane/utils"

type Master struct {
	ClusterList map[uint64]*Cluster
}

type ReplyData struct {
	ClusterId uint64 `json:"clusterId"`
}

type ProtoReplyMsg struct {
	ProtoId   uint64    `json:"protoId"`
	ReplyData ReplyData `json:"replyData"`
}

var InsaneMaster Master

func (master *Master) Init() {
	master.ClusterList = make(map[uint64]*Cluster)
}

func (master *Master) GenerateClusterId() uint64 {
	return uint64(utils.Now())
}

func (master *Master) AddCluster(cluster *Cluster) {
	master.ClusterList[cluster.ClusterId] = cluster
}
