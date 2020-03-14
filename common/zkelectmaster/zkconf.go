package zkelectmaster

import (
	"common/logger"
	"time"
)

var ElectionMaster *ElectionManager = &ElectionManager{
	nil,
	&ZookeeperConfig{},
	make(chan bool),
}

func ConnectZookeeper(isMasterChan chan bool, zkConfig *ZookeeperConfig){
	ElectionMaster = NewElectionManager(zkConfig, isMasterChan)
	go ElectionMaster.Run()
}
func OnNetworkChange(zkConfig *ZookeeperConfig)  {
	logger.InfoLog("begin choose new master,config: ",zkConfig.MasterPath)
	ElectionMaster.IsMasterQ <- false
	ElectionMaster.ZKClientConn.Close()
	time.Sleep(30*time.Second)
	ConnectZookeeper(ElectionMaster.IsMasterQ, zkConfig)
}