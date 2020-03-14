package reportstart

import (
	"common/report"
	"common/conf"
	"common/sqlitedb"
	"strconv"
	"common/logger"
	"time"
	"sync"
	"common/commonstruct"
	"fmt"
)
var stop = make(chan bool)
var stopkpi = make(chan bool)
var stopClean = make(chan bool)
var stopClean1 = make(chan bool)
// ReportKpiLock mutex lock.
var ReportKpiLock sync.Mutex
// ReportKpiStat kpi stat data.
var ReportKpiStat commonstruct.ReportKpi

func Start()  {
	logger.ErrorLog("report task start")
	var err error
	var timeSpan int
	var kpiSpan int

	timeSpan, err =strconv.Atoi(conf.Config["reporttimespan"])
	if err != nil{
		logger.ErrorLog(err)
		timeSpan = 5
	}
	kpiSpan, err =strconv.Atoi(conf.Config["kpitimespan"])
	if err != nil{
		logger.ErrorLog(err)
		kpiSpan = 60
	}
	go func() {
		for{
			select {
			case <-time.After(time.Duration(timeSpan)*time.Second):
				//状态上报
				reporttask,err := sqlitedb.Queryreportdb()
				if err != nil{
					logger.ErrorLog("get report task failed from reportdb, ",err)
					continue
				}
				reporttask.VirtualIp, _ = conf.Config["vip"]
				if reporttask.TaskNum != 0{
					err := report.StatusReport(reporttask)
					if err != nil{
						logger.ErrorLog("report failed, task: ",reporttask,", err: ", err)
						time.Sleep(15 * time.Second)
						continue
					}
					sqlitedb.Deletetaskdb(reporttask)
				}
			case <- stop:
				logger.InfoLog("report stop")
				return
			}
		}
	}()
	//清理上报失败任务
	go func(){
		for {
			select {
			case <-time.After(1* time.Minute):
				reporttask,err := sqlitedb.QueryOvertimereportdb()
				if err != nil{
					logger.ErrorLog("get overtime task failed from reportdb, ",err)
					continue
				}
				if reporttask.TaskNum != 0{
					err := sqlitedb.Deletetaskdb(reporttask)
					if err != nil{
						logger.ErrorLog("delete overtime task failed ",reporttask,", err: ", err)
						continue
					}
				}
				logger.InfoLog("delete overtime task success")
			case <-stopClean:
				logger.ErrorLog("task clean stop")
				return
			}
		}
	}()
	//清理超时没有任务
	go func(){
		for {
			select {
			case <-time.After(10* time.Minute):
				go func() {
					preheattask := sqlitedb.QueryOvertimePreheat()
					err := sqlitedb.Deletepreheatdb(preheattask)
					if err != nil{
						logger.ErrorLog("delete overtime preheattask failed", err)
					}
				}()
				refreshtask := sqlitedb.QueryOvertimeRefresh()
				err := sqlitedb.Deleterefreshdb(refreshtask)
				if err != nil{
					logger.ErrorLog("delete overtime refresh failed", err)
				}
				logger.InfoLog("delete overtime task success")
			case <-stopClean1:
				logger.ErrorLog("preheat and refresh task clean stop")
				return
			}
		}
	}()
	go func() {
		for{
			select {
			case <-time.After(time.Duration(kpiSpan)*time.Second):
				str := fmt.Sprintf("kpi stat, TotalTaskNum: %v, PreheatTaskNum: %v, PreheatSuccNum: %v, PreheatFailNum: %v, RefreshTaskNum: %v, RefreshSuccNum: %v, RefreshFailNum: %v",
					ReportKpiStat.TotalTaskNum, ReportKpiStat.PreheatTaskNum, ReportKpiStat.PreheatSuccNum, ReportKpiStat.PreheatFailNum, ReportKpiStat.RefreshTaskNum, ReportKpiStat.RefreshSuccNum, ReportKpiStat.RefreshFailNum)
				logger.ErrorLog(str)
			case <- stopkpi:
				logger.InfoLog("kpi stop")
				return
			}
		}
	}()
}


func Stop() {
	stop <- true
	stopkpi <- true
	stopClean <- true
	stopClean1 <- true
}