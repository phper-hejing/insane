package server

import (
	"encoding/json"
	"github.com/donnie4w/go-logger/logger"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"insane/utils"
	"math"
	"sync"
	"time"
)

const LISTEN_DURATION = 120

type ServerLoad struct {
	InitIoCounters net.IOCountersStat `json:"-"`
	Cpu            map[int64]uint32   `json:"cpu"`
	Mem            map[int64]uint32   `json:"mem"`
	Conn           map[int64]uint32   `json:"conn"`
	Io             map[int64]*IoInfo  `json:"io"`
	ServerInfo     ServerInfo         `json:"serverInfo"`
	M              sync.Mutex         `json:"-"`
}

type ServerInfo struct {
	Cpu uint32 `json:"cpu"`
	Mem uint32 `json:"mem"`
}

type IoInfo struct {
	Sent uint64 `json:"sent"`
	Recv uint64 `json:"recv"`
}

var InsaneLoad ServerLoad

func (serverLoad *ServerLoad) Init() error {
	serverLoad.GetServerInfo()
	serverLoad.Cpu = make(map[int64]uint32)
	serverLoad.Mem = make(map[int64]uint32)
	serverLoad.Conn = make(map[int64]uint32)
	serverLoad.Io = make(map[int64]*IoInfo)
	return nil
}

func (serverLoad *ServerLoad) Start(interval uint64) error {
	serverLoad.Init()
	var key int64 = 0
	step := 0
	keyAddr := make([]int64, LISTEN_DURATION)
	t := time.NewTicker(time.Duration(interval) * time.Second)
	for {
		<-t.C
		serverLoad.saveIoCounters()

		serverLoad.M.Lock()
		if step == LISTEN_DURATION {
			step = 0
		}

		key = utils.Now()
		if keyAddr[step] != 0 {
			delete(serverLoad.Cpu, keyAddr[step])
			delete(serverLoad.Mem, keyAddr[step])
			delete(serverLoad.Io, keyAddr[step])
			delete(serverLoad.Conn, keyAddr[step])
		}
		keyAddr[step] = key

		serverLoad.saveCpuLoad(keyAddr[step], serverLoad.getCpuLoad())
		serverLoad.saveMemLoad(keyAddr[step], serverLoad.getMemLoad())
		serverLoad.saveIoLoad(keyAddr[step], serverLoad.getIoLoad())
		serverLoad.saveConnNumber(keyAddr[step], serverLoad.getConn())

		serverLoad.M.Unlock()
		step++

	}
	return nil
}

func (serverLoad *ServerLoad) Get() (string, error) {
	serverLoad.M.Lock()
	data, err := json.Marshal(serverLoad)
	serverLoad.M.Unlock()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// 最近num个cpu的百分比
func (serverLoad *ServerLoad) GetLatelyCpuLoad(num int) (cpuLoad []uint32) {

	for i := 0; i < num; i++ {
		cpuLoad = append(cpuLoad, serverLoad.getCpuLoad())
	}
	return
}

func (serverLoad *ServerLoad) GetServerInfo() {
	cpuNum, _ := cpu.Counts(false)
	virtualMem, _ := mem.VirtualMemory()
	memSize := uint32(math.Ceil(float64(virtualMem.Total) / 1024 / 1024 / 1024))
	serverLoad.ServerInfo.Cpu = uint32(cpuNum)
	serverLoad.ServerInfo.Mem = memSize
}

func (serverLoad *ServerLoad) getCpuLoad() uint32 {
	c, err := cpu.Percent(time.Second*1, false)
	if err != nil {
		logger.Debug(err)
		return 0
	}
	return uint32(c[0])
}

func (serverLoad *ServerLoad) getMemLoad() uint32 {
	mem, err := mem.VirtualMemory()
	if err != nil {
		logger.Debug(err)
		return 0
	}
	return uint32(mem.UsedPercent)
}

func (serverLoad *ServerLoad) getIoLoad() *IoInfo {
	io, err := net.IOCounters(false)
	if err != nil {
		logger.Debug(err)
		return nil
	}
	initBytesSent := serverLoad.InitIoCounters.BytesSent
	initBytesRecv := serverLoad.InitIoCounters.BytesRecv
	curBytesSent := io[0].BytesSent
	curBytesRecv := io[0].BytesRecv
	sent := curBytesSent - initBytesSent
	recv := curBytesRecv - initBytesRecv
	return &IoInfo{
		Sent: sent,
		Recv: recv,
	}
}

func (serverLoad *ServerLoad) getConn() uint32 {
	conn, err := net.Connections("all")
	if err != nil {
		logger.Debug(err)
		return 0
	}
	return uint32(len(conn))
}

func (serverLoad *ServerLoad) saveIoCounters() error {
	ioc, err := net.IOCounters(false)
	if err != nil {
		logger.Debug(err)
		return err
	}
	serverLoad.InitIoCounters = ioc[0]
	return nil
}

func (serverLoad *ServerLoad) saveCpuLoad(key int64, value uint32) {
	serverLoad.Cpu[key] = value
}

func (serverLoad *ServerLoad) saveMemLoad(key int64, value uint32) {
	serverLoad.Mem[key] = value
}

func (serverLoad *ServerLoad) saveIoLoad(key int64, value *IoInfo) {
	serverLoad.Io[key] = value
}

func (serverLoad *ServerLoad) saveConnNumber(key int64, value uint32) {
	serverLoad.Conn[key] = value
}
