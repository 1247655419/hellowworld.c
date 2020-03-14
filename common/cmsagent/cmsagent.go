package cmsagent

import (
	//"database/sql"
	"common/sqlitedb"
	"common/conf"
	"strconv"
	"time"
	"common/logger"
	"common/parsefile"
	"reflect"
	"common/httpClient"
	"fmt"
)
var DevInfo = make(map[string]int)
var stop = make(chan bool)
func Init()  {
	logger.ErrorLog("watch config start")
	timeSpan,err := strconv.Atoi(conf.Config["getdevicetimespan"])
	if err != nil{
		logger.ErrorLog(err)
		timeSpan = 30
	}

	//t:=time.NewTicker(time.Duration(60*time.Second))
	go func() {
		for  {
			select {
			case <-time.After(time.Duration(timeSpan)*time.Second):
				oldDevinfo := sqlitedb.Querydevicedb()
				nowDevinfo,err := parsefile.Parsenetwork()
				if err != nil{
					continue
				}
				for i,devInfo:=range nowDevinfo.DevInfoList{
					nowDevinfo.DevInfoList[i].Stauts = httpClient.GetNodeStatus(devInfo.DevIp)
					str := fmt.Sprintf("get hcs nodeStatus, devip: [%s], nodeStatus: [%d]", devInfo.DevIp, nowDevinfo.DevInfoList[i].Stauts)
					logger.InfoLog(str)
				}
				if oldDevinfo.PopId != nowDevinfo.PopId{
					sqlitedb.Insertdevicedb(nowDevinfo)
					continue
				}
				if oldDevinfo.GroupiId != nowDevinfo.GroupiId{
					sqlitedb.Insertdevicedb(nowDevinfo)
					continue
				}
				if !reflect.DeepEqual(oldDevinfo.DevInfoList,nowDevinfo.DevInfoList){
					sqlitedb.Insertdevicedb(nowDevinfo)
					continue
				}
			case <- stop:
				logger.ErrorLog("get groupInfo stop")
				return
			}
		}
	}()
}
func Stop(){
	stop  <- true
}


	