package main

/*
#cgo CFLAGS: -I./service_agent/include
#cgo CFLAGS: -fstack-protector-all -fPIE -pie
#cgo LDFLAGS: -Wl,-z,now
#cgo LDFLAGS: -Wl,-z,relro
#cgo LDFLAGS: -Wl,--disable-new-dtags,--rpath="/home/icache/hcs/hcs.nginx/lib"
#cgo LDFLAGS: -L./service_agent/lib/ -lcms_agent
#include "cmsagent_interface.h"
*/
import "C"

import (
	"common/cmsagent"
	"common/logger"
	"common/parsefile"
	"common/sqlitedb"
	"common/zkelectmaster"
	"fmt"
	"os"
	"preheatmcserv/getstart"
	"preheatmcserv/putstart"
	"preheatmcserv/reportstart"
	"strconv"
	"time"
)

func main() {
	defer func() {
		if errs := recover(); errs != nil {
			logger.ErrorLog("CMSAgent runtime err ", errs)
		}
		time.Sleep(time.Duration(2) * time.Second)
		//关闭日志
		logger.LoggerShutDown()
		os.Exit(-1)
	}()

	err, zkIp := parsefile.ParseXml()
	if err != nil{
		logger.ErrorLog("init zkIp error, ",err)
		return
	}
	nowDevinfo,err := parsefile.Parsenetwork()
	if err != nil{
		logger.ErrorLog("init deviceInfo error, ",err)
		return
	}
	err = sqlitedb.Init()
	if err != nil {
		logger.ErrorLog("init sqlitedb error, ",err)
		return
	}
	defer sqlitedb.Closedb()

	go logger.StartEasyServer()
	C.SetLogLevelInit()

	masterPath := "/master_" + strconv.Itoa(nowDevinfo.GroupiId) + strconv.Itoa(nowDevinfo.PopId)
	zkConfig := &zkelectmaster.ZookeeperConfig{
		Servers:    zkIp,
		RootPath:   "/cmsAgent",
		MasterPath: masterPath,
	}
	// main goroutine 和 选举goroutine之间通信的channel，同于返回选角结果
	//isMasterChan := make(chan bool)
	var isMaster bool
	// 选举
	zkelectmaster.ConnectZookeeper(zkelectmaster.ElectionMaster.IsMasterQ, zkConfig)
	//标志位
	var flag  = false
	for {
		select {
		case isMaster = <-zkelectmaster.ElectionMaster.IsMasterQ:
			if isMaster {
				// do some job on master
				fmt.Println("begin start")
				flag = true
				//初始化
				cmsagent.Init()
				//启动任务拉取
				getstart.Start()
				//启动状态上报
				reportstart.Start()
				//启动下发任务
				putstart.Flag = true
				putstart.Start()

			}else {
				if flag == true{
					//停止获取分组信息
					fmt.Println("begin stop")
					flag = false
					cmsagent.Stop()
					//停止任务拉取
					getstart.Stop()
					//停止状态上报
					reportstart.Stop()
					//停止下发任务
					putstart.Stop()
				}
			}
		}
	}

}
