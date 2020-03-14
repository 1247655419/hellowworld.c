package getstart

import (
	"common/conf"
	"common/gettask"
	"common/logger"
	"common/sqlitedb"
	"fmt"
	"strconv"
	"time"
)
var stop = make(chan bool)
func Start()  {
	logger.ErrorLog("get task start")
	timeSpan, err :=strconv.Atoi(conf.Config["gettasktimespan"])
	if err != nil{
		logger.ErrorLog(err)
		timeSpan = 5
	}
	go func() {
		for{
			select {
			case <-time.After(time.Duration(timeSpan)*time.Second):
				//后续增加判断任务状态
				optimalDev,err := sqlitedb.GetBestDevice()
				if err != nil {
					logger.ErrorLog("get bestHCSdevice failed, ",err)
					time.Sleep(5*time.Second)
					continue
				}
				if optimalDev.DevInfoList[0].Stauts > 5{
					str := fmt.Sprintf("dstip: [%s] nodestatus is too highm nodestatus: [%d]", optimalDev.DevInfoList[0].DevIp, optimalDev.DevInfoList[0].Stauts)
					logger.ErrorLog(str)
					time.Sleep(10*time.Second)
					continue
				}
				dataNum := sqlitedb.QueryCount()
				if dataNum > 40960{
					logger.ErrorLog("db is fulled")
					time.Sleep(20*time.Second)
					continue
				}
				err = gettask.GetTaskfromHTTP()
				if err!=nil{
					logger.ErrorLog("get task failed, ",err)
					continue
				}
				logger.InfoLog("get task success")
			case <- stop:
				logger.ErrorLog("get task stop")
				return
			}
		}
	}()
}
func Stop()  {
	stop <- true
}