package putstart

import (
	"common/commonstruct"
	"common/conf"
	"common/logger"
	"common/parsefile"
	"common/puttask"
	"common/sqlitedb"
	"fmt"
	"os"
	"strconv"
	"time"
	"preheatmcserv/reportstart"
)
var start =make(chan bool)
var Flag bool
var stop = make(chan bool)
func Start()  {
	logger.ErrorLog("put task start")
	localid, localip := parsefile.GetlocalId()
	if localip == "" || localid == 0{
		str := fmt.Sprintf("get localip, localid failed, localip: %s, localid: %d",localip, localid)
		logger.ErrorLog(str)
		os.Exit(-1)
	}
	//与NGX最大连接数
	timeSleep,err :=   strconv.Atoi(conf.Config["timesleep"])
	if err != nil{
		logger.ErrorLog(err)
		timeSleep = 1
	}
	maxrefreshqueue, err := strconv.Atoi(conf.Config["maxrefreshqueue"])
	if err != nil{
		logger.ErrorLog(err)
		maxrefreshqueue = 200
	}
	maxpreheatqueue, err := strconv.Atoi(conf.Config["maxpreheatqueue"])
	if err != nil{
		logger.ErrorLog(err)
		maxpreheatqueue = 100
	}
	refreshRetryTimes, err := strconv.Atoi(conf.Config["refreshretrytimes"])
	if err != nil{
		logger.ErrorLog(err)
		refreshRetryTimes = 3
	}
	preheatRetryTimes, err := strconv.Atoi(conf.Config["preheatretrytimes"])
	if err != nil{
		logger.ErrorLog(err)
		preheatRetryTimes = 3
	}
	go func() {
		for {
			if Flag == false{
				logger.ErrorLog("put task stop")
				return
			}
			count := sqlitedb.QueryDeviceCount()
			if count == 0 {
				logger.ErrorLog("no device")
				time.Sleep(5*time.Second)
				continue
			}
			time.Sleep(time.Duration(timeSleep)*time.Millisecond)//防止CPU过高
			if puttask.RefreshGoroutines < maxrefreshqueue*count{
				//本分组信息
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
				//nodeStatus := httpClient.GetNodeStatus(optimalDev.DevInfoList[0].DevIp)
				//if nodeStatus > 5{
				//	sqlitedb.Updatedevicedb(nodeStatus, optimalDev.DevInfoList[0].DevIp)
				//}
				//刷新任务
				fmt.Println("refresh1",time.Now().Unix())
				refreshtask,err :=sqlitedb.Queryrefreshdb()
				if err != nil{
					logger.ErrorLog("get refreshtask failed, ",err)
					continue
				}
				fmt.Println("refresh2",time.Now().Unix())
				if refreshtask.Num != 0{
					for i:=0;i<refreshtask.Num;i++{
						subrefreshtask := refreshtask.RefreshTask[i]
						subrefreshtask.ReqRefreshTask[0].DevId = strconv.Itoa(optimalDev.DevInfoList[0].DevId)//strconv.Itoa(localid)
						subrefreshtask.ReqRefreshTask[0].GroupId = strconv.Itoa(optimalDev.GroupiId)
						//logger.InfoLog("connection number: ",puttask.RefreshGoroutines)
						go func() {
							var reportstatus commonstruct.ReportStatus
							var err error
							if 	subrefreshtask.ReqRefreshTask[0].UrlType == "3"{
								reportstatus,err = puttask.SendMenuRefreshToNgxNew(subrefreshtask)
							}else {
								reportstatus,err = puttask.SendRefreshToNgxNew(subrefreshtask,optimalDev.DevInfoList[0].DevIp)
							}
							if err != nil{
								if subrefreshtask.RetryTimes > refreshRetryTimes-2{
									sqlitedb.UpdateOrInsertreportdb(reportstatus)

									reportstart.ReportKpiLock.Lock()
									reportstart.ReportKpiStat.RefreshFailNum++
									reportstart.ReportKpiLock.Unlock()

									str := fmt.Sprintf("taskid: [%d], result: [%d], err_code: [%d], err_msg: [%s], start_time: [%d], end_time[%d], tasktype: [refresh], Url: [%s], dstaddr: [%s]",
										reportstatus.SubTaskId, reportstatus.Status, reportstatus.Err_code, reportstatus.Err_Msg, reportstatus.Start_time, reportstatus.End_time, reportstatus.Url,
										optimalDev.DevInfoList[0].DevIp)
									logger.RecordLog(str)
									//sqlitedb.Updaterefreshdb(subrefreshtask.RetryTimes+1, 2, subrefreshtask.ReqRefreshTask[0].TaskId)
									return
								}
								sqlitedb.Updaterefreshdb(subrefreshtask.RetryTimes+1, 0, subrefreshtask.ReqRefreshTask[0].TaskId)
								return
							}else {
								sqlitedb.UpdateOrInsertreportdb(reportstatus)

								reportstart.ReportKpiLock.Lock()
								reportstart.ReportKpiStat.RefreshSuccNum++
								reportstart.ReportKpiLock.Unlock()

								str := fmt.Sprintf("taskid: [%d], result: [%d], err_code: [%d], err_msg: [%s], start_time: [%d], end_time[%d], tasktype: [refresh], Url: [%s], dstaddr: [%s]",
									reportstatus.SubTaskId, reportstatus.Status, reportstatus.Err_code, reportstatus.Err_Msg, reportstatus.Start_time, reportstatus.End_time, reportstatus.Url,
									optimalDev.DevInfoList[0].DevIp)
								logger.RecordLog(str)
								//sqlitedb.Updaterefreshdb(subrefreshtask.RetryTimes, 2, subrefreshtask.ReqRefreshTask[0].TaskId)
								fmt.Println("time:",time.Now().Unix(),",taskid:",subrefreshtask.ReqRefreshTask[0].TaskId)
								return
							}

						}()
					}
				}else {
					//预热启动标志
					preheatNum := sqlitedb.QueryPreheatCount()
					if preheatNum == 0{
						logger.ErrorLog("preheattask is NULL")
						time.Sleep(5*time.Second)
						continue
					}
					start<-true
				}
				fmt.Println("refresh3",time.Now().Unix())
			}
		}
	}()
	go func() {
		for {
			select {
			case <-start:
				count := sqlitedb.QueryDeviceCount()
				if count == 0 {
					logger.ErrorLog("no device")
					time.Sleep(5*time.Second)
					continue
				}
				if puttask.PreheatGoroutines < maxpreheatqueue*count{
					//本分组信息
					optimalDev,err := sqlitedb.GetBestDevice()
					if err != nil {
						logger.ErrorLog("get bestHCSdevice failed, ",err)
						time.Sleep(5*time.Second)
						continue
					}
					if optimalDev.DevInfoList[0].Stauts > 5{
						str := fmt.Sprintf("dstip: [%d] nodestatus is too highm nodestatus: [%d]", optimalDev.DevInfoList[0].DevIp,
							optimalDev.DevInfoList[0].Stauts)
						logger.ErrorLog(str)
						time.Sleep(10*time.Second)
						continue
					}
					//nodeStatus := httpClient.GetNodeStatus(optimalDev.DevInfoList[0].DevIp)
					//if nodeStatus > 5{
					//	sqlitedb.Updatedevicedb(nodeStatus, optimalDev.DevInfoList[0].DevIp)
					//}
					tmppreheattask,err := sqlitedb.Querypreheatdb()
					if err!=nil{
						logger.ErrorLog("get preheattask failed, ",err)
						continue
					}
					var preheattask commonstruct.DbTempReqPreheatStruct
					for i:=0;i<tmppreheattask.Num;i++ {
						subpreheattask := tmppreheattask.PreheatTask[i]
						subpreheattask.ReqPreheatTask[0].SrcDeviceId = localid
						subpreheattask.ReqPreheatTask[0].SrcDeviceIp = localip
						//去重
						sqlitedb.TaskQueueLock.Lock()
						err := sqlitedb.InsertWaitTaskQueue(subpreheattask)
						sqlitedb.TaskQueueLock.Unlock()
						if err != nil{
							logger.ErrorLog(err)
							continue
						}
						preheattask.Num++
						preheattask.PreheatTask = append(preheattask.PreheatTask, subpreheattask)
					}
					for i:=0;i<preheattask.Num;i++ {
						subpreheattask := preheattask.PreheatTask[i]
						go func() {
							reportstatus,err := puttask.SendPreheatToNgxNew(subpreheattask,optimalDev.DevInfoList[0].DevIp)
							if err != nil{
								sqlitedb.TaskQueueLock.Lock()
								val, _ := sqlitedb.WaitPreheatTask.Get(subpreheattask.ReqPreheatTask[0].ResUrl)
								//任务结束删除队列任务
								sqlitedb.WaitPreheatTask.Remove(subpreheattask.ReqPreheatTask[0].ResUrl)
								sqlitedb.TaskQueueLock.Unlock()
								if val != nil {
									switch val.(type) {
									case commonstruct.TempReqPreheatStruct:
										if subpreheattask.RetryTimes > preheatRetryTimes-2 { //达到失败次数，不再下发
											//sqlitedb.Updatepreheatdb(subpreheattask.RetryTimes+1, 2, subpreheattask.ReqPreheatTask[0].TaskId)
											sqlitedb.UpdateOrInsertreportdb(reportstatus)

											reportstart.ReportKpiLock.Lock()
											reportstart.ReportKpiStat.PreheatFailNum++
											reportstart.ReportKpiLock.Unlock()

											str := fmt.Sprintf("taskid: [%d], result: [%d], err_code: [%d], err_msg: [%s], start_time: [%d], end_time[%d], tasktype: [inject], Url: [%s], dstaddr: [%s]",
												reportstatus.SubTaskId, reportstatus.Status, reportstatus.Err_code, reportstatus.Err_Msg, reportstatus.Start_time, reportstatus.End_time, reportstatus.Url,
												optimalDev.DevInfoList[0].DevIp)
											logger.RecordLog(str)
										}else {
											sqlitedb.Updatepreheatdb(subpreheattask.RetryTimes+1, 0,  subpreheattask.ReqPreheatTask[0].TaskId)
										}
										return
									case []commonstruct.TempReqPreheatStruct:
										for _, task := range val.([]commonstruct.TempReqPreheatStruct) {
											if task.RetryTimes > preheatRetryTimes-2 { //达到失败次数，不再下发
												//sqlitedb.Updatepreheatdb(task.RetryTimes+1, 2, task.ReqPreheatTask[0].TaskId)
												reportstatus.SubTaskId, _ = strconv.Atoi(task.ReqPreheatTask[0].TaskId)
												sqlitedb.UpdateOrInsertreportdb(reportstatus)

												reportstart.ReportKpiLock.Lock()
												reportstart.ReportKpiStat.PreheatFailNum++
												reportstart.ReportKpiLock.Unlock()

												str := fmt.Sprintf("taskid: [%d], result: [%d], err_code: [%d], err_msg: [%s], start_time: [%d], end_time[%d], tasktype: [inject], Url: [%s], dstaddr: [%s]",
													reportstatus.SubTaskId, reportstatus.Status, reportstatus.Err_code, reportstatus.Err_Msg, reportstatus.Start_time, reportstatus.End_time, reportstatus.Url,
													optimalDev.DevInfoList[0].DevIp)
												logger.RecordLog(str)
											}else {
												sqlitedb.Updatepreheatdb(task.RetryTimes+1, 0,  task.ReqPreheatTask[0].TaskId)
											}
										}
										return
									default:
										logger.ErrorLog("data type is unknown")
									}
								}
							}
						}()
					}
				}
			case <-stop:
				logger.ErrorLog("put preheattask stop")
				return
			}
		}
	}()
}
func Stop(){
	Flag = false
	stop <- true
}
