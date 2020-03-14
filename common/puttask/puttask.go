package puttask

import (
	"bufio"
	"common/commonstruct"
	"common/conf"
	"common/httpClient"
	"common/logger"
	"common/sqlitedb"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"preheatmcserv/reportstart"
)

var (
	refreshRetryTime, _ = strconv.Atoi(conf.Config["retrytimespan"])
	preheatRetryTime, _ = strconv.Atoi(conf.Config["retrytimespan"])
)

//刷新状态结果
const (
	Resource_Not_Found = -2
	Delete_Fail        = -1
	Delete_Success     = 0
)

//预热状态结果
const (
	Preheat_Failed = -1
	Completed      = 0
	Downloading    = 2
	Cached_hit     = 4
)

//上报状态结果
const (
	SUCCESS = 3
	FALIED  = 4
)

var RefreshGoroutines int
var PreheatGoroutines int
var preheatmu sync.Mutex // 互斥锁
var refreshmu sync.Mutex // 互斥锁

func SendRefreshToNgx(subrefreshtask commonstruct.TempReqRefreshStruct, devip string) (commonstruct.ReportStatus, error) {
	refreshmu.Lock()
	RefreshGoroutines++
	refreshmu.Unlock()
	defer func() {
		if errs := recover(); errs != nil {
			logger.ErrorLog("CMSAgent runtime err, panic: ", errs)
			sqlitedb.Updaterefreshdb(0, 0, subrefreshtask.ReqRefreshTask[0].TaskId)
		}
		refreshmu.Lock()
		RefreshGoroutines--
		refreshmu.Unlock()
		return
	}()
	refreshtask := subrefreshtask.ReqRefreshTask
	//发送请求
	var start_time int64
	var end_time int64
	var err error
	var ngxReport commonstruct.ResqRefreshStruct
	var reportstatus commonstruct.ReportStatus
	reportstatus.Url = refreshtask[0].ResUrl
	reportstatus.SubTaskId, err = strconv.Atoi(refreshtask[0].TaskId)
	if err != nil {
		logger.ErrorLog(err)
	}
	reportstatus.Level = subrefreshtask.Level
	//转换为json格式下发
	task, err := json.Marshal(refreshtask)
	if err != nil {
		logger.ErrorLog("json convert err", err)
		reportstatus.Status = FALIED
		reportstatus.Err_Msg = "over retrytimes"
		reportstatus.Err_code = http.StatusInternalServerError
		return reportstatus, err
	}

	start_time = time.Now().Unix()
	resp, err := httpClient.SendRefreshTask(devip, task)
	if err != nil {
		end_time = time.Now().Unix()
		str := fmt.Sprintf("put refreshtask to device: [%s] fail, task: [%s], url: [%s], failed times: [%d],  connectionnum: [%d]",
			devip, refreshtask[0].TaskId, refreshtask[0].ResUrl, subrefreshtask.RetryTimes+1, RefreshGoroutines)
		logger.ErrorLog(str, err)
		reportstatus.Status = FALIED
		reportstatus.Err_Msg = "over retrytimes"
		reportstatus.Start_time = start_time
		reportstatus.End_time = end_time
		reportstatus.Err_code = http.StatusInternalServerError
		return reportstatus, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		end_time = time.Now().Unix()
		if err != nil {
			str := fmt.Sprintf("put refreshtask to device: [%s] fail, task: [%s], url: [%s], failed times: [%d], connectionnum: [%d]",
				devip, refreshtask[0].TaskId, refreshtask[0].ResUrl, subrefreshtask.RetryTimes+1, RefreshGoroutines)
			logger.ErrorLog(str, err)
			reportstatus.Status = FALIED
			reportstatus.Err_Msg = "over retrytimes"
			reportstatus.Start_time = start_time
			reportstatus.End_time = end_time
			reportstatus.Err_code = http.StatusInternalServerError
			return reportstatus, err
		}
		str := fmt.Sprintf("get refreshResp from Ngx, RcvMsg:%s, taskid:[%s]", string(body), subrefreshtask.ReqRefreshTask[0].TaskId)
		logger.ErrorLog(str)
		err = json.Unmarshal(body, &ngxReport)
		if err != nil {
			logger.ErrorLog(err)
		}
		if ngxReport.Result != Resource_Not_Found && ngxReport.Result != Delete_Success {
			str := fmt.Sprintf("put refreshtask to device: [%s] fail, task: [%s], url: [%s], failed times: [%d], connectionnum: [%d]",
				devip, refreshtask[0].TaskId, refreshtask[0].ResUrl, subrefreshtask.RetryTimes+1, RefreshGoroutines)
			logger.ErrorLog(str)
			reportstatus.Status = FALIED
			reportstatus.Err_Msg = ngxReport.ResultMsg
			reportstatus.Start_time = start_time
			reportstatus.End_time = end_time
			reportstatus.Err_code = ngxReport.Result
			return reportstatus, errors.New(ngxReport.ResultMsg)
		}
		reportstatus.Status = SUCCESS
		reportstatus.Err_Msg = ngxReport.ResultMsg
		reportstatus.Progress = 100
		reportstatus.Start_time = start_time
		reportstatus.End_time = end_time
		reportstatus.Err_code = http.StatusOK
		return reportstatus, nil
	} else if resp.StatusCode == http.StatusNotFound {
		end_time = time.Now().Unix()
		reportstatus.Status = SUCCESS
		reportstatus.Start_time = start_time
		reportstatus.End_time = end_time
		reportstatus.Err_code = http.StatusNotFound
		reportstatus.Progress = 100
		reportstatus.Err_Msg = "DELETE_SUCCESS"
		logger.InfoLog("connectionnum: ", RefreshGoroutines)
		return reportstatus, nil
	} else {
		end_time = time.Now().Unix()
		reportstatus.Status = FALIED
		reportstatus.Err_Msg = "over retrytimes"
		reportstatus.Start_time = start_time
		reportstatus.End_time = end_time
		reportstatus.Err_code = resp.StatusCode
		str := fmt.Sprintf("put refreshtask to device: [%s] fail, task: [%s], failed times: [%d], err_code: [%d], connectionnum: %d",
			devip, refreshtask[0].TaskId, subrefreshtask.RetryTimes+1, resp.StatusCode, RefreshGoroutines)
		logger.ErrorLog(str)
		return reportstatus, errors.New(strconv.Itoa(resp.StatusCode))

	}
	return reportstatus, nil
}
func SendRefreshToNgxNew(subrefreshtask commonstruct.TempReqRefreshStruct, devip string) (commonstruct.ReportStatus, error) {
	var reportstatus1 commonstruct.ReportStatus
	var err1 error
	var reportstatus2 commonstruct.ReportStatus
	var err2 error
	var resurl = subrefreshtask.ReqRefreshTask[0].ResUrl

	times := refreshRetryTime
	if subrefreshtask.RetryTimes != 0 {
		time.Sleep(time.Duration(times) * time.Second)
	}

	reportstatus1, err1 = SendRefreshToNgx(subrefreshtask, devip)
	if len(subrefreshtask.AccessUrl) > 0 {
		subrefreshtask.ReqRefreshTask[0].ResUrl = subrefreshtask.AccessUrl
		reportstatus2, err2 = SendRefreshToNgx(subrefreshtask, devip)
		subrefreshtask.ReqRefreshTask[0].ResUrl = resurl
	}

	if err1 != nil {
		return reportstatus1, err1
	}
	if err2 != nil {
		return reportstatus2, err2
	}
	return reportstatus1, err1
}
func SendMenuRefreshToNgx(subrefreshtask commonstruct.TempReqRefreshStruct) (commonstruct.ReportStatus, error) {
	Devinfo := sqlitedb.Querydevicedb()
	var reports []commonstruct.ReportStatus
	var errs []error
	for _, devinfo := range Devinfo.DevInfoList {
		logger.InfoLog("destIP: ", devinfo.DevIp)
		subrefreshtask.ReqRefreshTask[0].DevId = strconv.Itoa(devinfo.DevId) //strconv.Itoa(localid)
		report, err := SendRefreshToNgx(subrefreshtask, devinfo.DevIp)
		reports = append(reports, report)
		errs = append(errs, err)
	}
	for i := range errs {
		if errs[i] != nil {
			return reports[i], errs[i]
		}
	}

	return reports[0], nil
}
func SendMenuRefreshToNgxNew(subrefreshtask commonstruct.TempReqRefreshStruct) (commonstruct.ReportStatus, error) {
	var reportstatus1 commonstruct.ReportStatus
	var err1 error
	var reportstatus2 commonstruct.ReportStatus
	var err2 error
	var resurl = subrefreshtask.ReqRefreshTask[0].ResUrl

	times := refreshRetryTime
	if subrefreshtask.RetryTimes != 0 {
		time.Sleep(time.Duration(times) * time.Second)
	}

	reportstatus1, err1 = SendMenuRefreshToNgx(subrefreshtask)
	if len(subrefreshtask.AccessUrl) > 0 {
		subrefreshtask.ReqRefreshTask[0].ResUrl = subrefreshtask.AccessUrl
		reportstatus2, err2 = SendMenuRefreshToNgx(subrefreshtask)
		subrefreshtask.ReqRefreshTask[0].ResUrl = resurl
	}

	if err1 != nil {
		return reportstatus1, err1
	}
	if err2 != nil {
		return reportstatus2, err2
	}
	return reportstatus1, err1
}
func SendPreheatToNgx(subpreheattask commonstruct.TempReqPreheatStruct, devip string) (commonstruct.ReportStatus, error) {
	preheatmu.Lock()
	PreheatGoroutines++
	preheatmu.Unlock()
	defer func() {
		if errs := recover(); errs != nil {
			logger.ErrorLog("CMSAgent runtime err, panic: ", errs)
			subpreheattask.ReqPreheatTask[0].ResUrl = subpreheattask.OriResUrl
			sqlitedb.TaskQueueLock.Lock()
			val, _ := sqlitedb.WaitPreheatTask.Get(subpreheattask.ReqPreheatTask[0].ResUrl)
			sqlitedb.WaitPreheatTask.Remove(subpreheattask.ReqPreheatTask[0].ResUrl)
			sqlitedb.TaskQueueLock.Unlock()
			if val != nil {
				switch val.(type) {
				case commonstruct.TempReqPreheatStruct:
					sqlitedb.Updatepreheatdb(0, 0, subpreheattask.ReqPreheatTask[0].TaskId)
				case []commonstruct.TempReqPreheatStruct:
					for _, task := range val.([]commonstruct.TempReqPreheatStruct) {
						sqlitedb.Updatepreheatdb(0, 0, task.ReqPreheatTask[0].TaskId)
					}
				default:
					logger.ErrorLog("data type is unknow")
				}
			}

		}
		preheatmu.Lock()
		PreheatGoroutines--
		preheatmu.Unlock()
		return
	}()
	preheattask := subpreheattask.ReqPreheatTask

	//发送请求
	var start_time int64
	var end_time int64
	var err error
	var reportstatus commonstruct.ReportStatus
	reportstatus.Url = preheattask[0].ResUrl
	reportstatus.SubTaskId, err = strconv.Atoi(preheattask[0].TaskId)
	if err != nil {
		logger.ErrorLog(err)
	}
	reportstatus.Level = subpreheattask.Level
	//转换为json格式下发
	task, err := json.Marshal(preheattask)
	if err != nil {
		logger.ErrorLog("json convert err", err)
		reportstatus.Status = FALIED
		reportstatus.Err_Msg = "over retrytimes"
		reportstatus.Err_code = http.StatusInternalServerError
		return reportstatus, err
	}

	start_time = time.Now().Unix()
	resp, err := httpClient.SendPreheatTask(devip, task)
	if err != nil {
		end_time = time.Now().Unix()
		str := fmt.Sprintf("put preheatTask to device: [%s] fail, task: [%s], url: [%s], failed times: [%d]",
			devip, preheattask[0].TaskId, preheattask[0].ResUrl, subpreheattask.RetryTimes+1)
		logger.ErrorLog(str)
		reportstatus.Status = FALIED
		reportstatus.Err_Msg = "over retrytimes"
		reportstatus.Start_time = start_time
		reportstatus.End_time = end_time
		reportstatus.Err_code = http.StatusInternalServerError
		return reportstatus, err
	}
	defer resp.Body.Close()
	if preheattask[0].UrlType == "5" {
		//处理m3u8
		//err = processM3u8(resp, subpreheattask, devip)
		//if err != nil{
		//	str := fmt.Sprintf("put preheattask  fail, task: [%s], url: [%s], failtimes: [%d]",
		//		subpreheattask.ReqPreheatTask[0].TaskId, subpreheattask.ReqPreheatTask[0].ResUrl, i+1)
		//	logger.InfoLog(str, err)
		//	continue
		//}
		return reportstatus, nil
	} else {
		//处理URL
		err, code, err_msg := processUrl(resp, subpreheattask, devip)
		end_time = time.Now().Unix()
		if err == nil {
			logger.InfoLog("connectionnum: ", PreheatGoroutines)
			return reportstatus, nil
		}
		str := fmt.Sprintf("put preheattask  fail, task: [%s], url: [%s], failtimes: [%d], connectionNum: %d",
			subpreheattask.ReqPreheatTask[0].TaskId, subpreheattask.ReqPreheatTask[0].ResUrl, subpreheattask.RetryTimes+1, PreheatGoroutines)
		logger.ErrorLog(str, err)
		reportstatus.Status = FALIED
		reportstatus.Err_Msg = err_msg
		reportstatus.Start_time = start_time
		reportstatus.End_time = end_time
		reportstatus.Err_code = code
		return reportstatus, err
	}
	return reportstatus, nil
}
func SendPreheatToNgxNew(subpreheattask commonstruct.TempReqPreheatStruct, devip string) (commonstruct.ReportStatus, error) {
	var reportstatus1 commonstruct.ReportStatus
	var err1 error
	var reportstatus2 commonstruct.ReportStatus
	var err2 error
	var AccessUrl = subpreheattask.ReqPreheatTask[0].AccessUrl

	subpreheattask.ReqPreheatTask[0].AccessUrl = ""
	subpreheattask.OriResUrl = subpreheattask.ReqPreheatTask[0].ResUrl

	times := preheatRetryTime
	if subpreheattask.RetryTimes != 0 {
		time.Sleep(time.Duration(times) * time.Second)
	}

	reportstatus1, err1 = SendPreheatToNgx(subpreheattask, devip)
	if len(AccessUrl) > 0 {
		subpreheattask.ReqPreheatTask[0].ResUrl = AccessUrl
		reportstatus2, err2 = SendPreheatToNgx(subpreheattask, devip)
		subpreheattask.ReqPreheatTask[0].ResUrl = subpreheattask.OriResUrl
		if err2 == nil {
			return reportstatus2, nil
		}
	}

	return reportstatus1, err1

}

//处理预热URL
func processUrl(resp *http.Response, subpreheattask commonstruct.TempReqPreheatStruct, devip string) (error, int, string) {
	var code int
	var err_msg string
	task, err := json.Marshal(subpreheattask.ReqPreheatTask)
	if err != nil {
		logger.ErrorLog(err)
	}
	start_time := time.Now().Unix()
	var ngxReport commonstruct.RespPreheatStruct
	var reportstatus commonstruct.ReportStatus
	if resp.StatusCode == http.StatusOK { //如果响应码为200
		reader := bufio.NewReader(resp.Body)
		var tmpStr string
		flag := false
		for {
			//reader.
			line, err := reader.ReadByte()
			if err != nil {
				str := fmt.Sprintf("put preheattask fail, task: [%s], url: [%s], err_code: ",
					subpreheattask.ReqPreheatTask[0].TaskId, subpreheattask.ReqPreheatTask[0].ResUrl)
				logger.ErrorLog(str, resp.StatusCode, ", ", err)
				code = http.StatusInternalServerError
				err_msg = "over retrytimes"
				return err, code, err_msg
			}
			if string(line) == "{" {
				tmpStr = tmpStr + string(line)
				flag = true
				continue
			}
			if string(line) != "}" && flag {
				tmpStr = tmpStr + string(line)
				continue
			}
			if string(line) == "}" {
				flag = false
				tmpStr = tmpStr + string(line)
				end_time := time.Now().Unix()
				str := fmt.Sprintf("get preheatResp from Ngx, RcvMsg:%s, taskid:[%s]", tmpStr, subpreheattask.ReqPreheatTask[0].TaskId)
				logger.ErrorLog(str)
				err := json.Unmarshal([]byte(tmpStr), &ngxReport)
				if err != nil {
					//logger.ErrorLog(str)
				}
				if ngxReport.Result != Completed && ngxReport.Result != Downloading && ngxReport.Result != Cached_hit {
					err = errors.New("failed")
					code = ngxReport.Result
					err_msg = ngxReport.ResultMsg
					return errors.New("failed"), code, err_msg
				}
				tmpStr = ""
				reportstatus.SubTaskId, err = strconv.Atoi(subpreheattask.ReqPreheatTask[0].TaskId)
				if err != nil {
					logger.ErrorLog("taskid type error, ", err)
				}
				reportstatus.Status = ngxReport.Result
				cachedSize := ngxReport.CachedSize
				if ngxReport.ResSize == "" {
					reportstatus.Progress = 0
				} else {
					resSize, err := strconv.Atoi(ngxReport.ResSize)
					if err != nil {
						logger.ErrorLog(err)
						resSize = 0
					}
					if resSize == 0 {
						reportstatus.Progress = 0
					} else {
						reportstatus.Progress = cachedSize * 100 / resSize
					}
				}
				reportstatus.Start_time = start_time
				reportstatus.End_time = end_time
				reportstatus.Level = subpreheattask.Level
				reportstatus.Url = subpreheattask.ReqPreheatTask[0].ResUrl
				reportstatus.Err_code = http.StatusOK
				reportstatus.Err_Msg = ngxReport.ResultMsg
				if ngxReport.Result == Completed || ngxReport.Result == Cached_hit {
					err = QueryTask(subpreheattask, ngxReport, devip)
					if err != nil {
						code = resp.StatusCode
						err_msg = "md5 checkout failed"
						return errors.New("md5 checkout failed"), code, err_msg
					}
					reportstatus.Status = SUCCESS
					reportstatus.Progress = 100
					//MD5查询
					subpreheattask.ReqPreheatTask[0].ResUrl = subpreheattask.OriResUrl
					sqlitedb.TaskQueueLock.Lock()
					val, _ := sqlitedb.WaitPreheatTask.Get(subpreheattask.ReqPreheatTask[0].ResUrl)
					sqlitedb.WaitPreheatTask.Remove(subpreheattask.ReqPreheatTask[0].ResUrl)
					sqlitedb.TaskQueueLock.Unlock()
					if val != nil {
						switch val.(type) {
						case commonstruct.TempReqPreheatStruct:
							//sqlitedb.Updatepreheatdb(subpreheattask.RetryTimes,2, subpreheattask.ReqPreheatTask[0].TaskId)
							sqlitedb.UpdateOrInsertreportdb(reportstatus)

							reportstart.ReportKpiLock.Lock()
							reportstart.ReportKpiStat.PreheatSuccNum++
							reportstart.ReportKpiLock.Unlock()

							str := fmt.Sprintf("taskid: [%d], result: [%d], err_code: [%d], err_msg: [%s], start_time: [%d], end_time[%d], tasktype: [inject], Url: [%s], dstaddr: [%s]",
								reportstatus.SubTaskId, reportstatus.Status, reportstatus.Err_code, reportstatus.Err_Msg, reportstatus.Start_time, reportstatus.End_time, reportstatus.Url, devip)
							logger.RecordLog(str)
							return nil, code, err_msg
						case []commonstruct.TempReqPreheatStruct:
							for _, task := range val.([]commonstruct.TempReqPreheatStruct) {
								//sqlitedb.Updatepreheatdb(task.RetryTimes,2, task.ReqPreheatTask[0].TaskId)
								reportstatus.SubTaskId, _ = strconv.Atoi(task.ReqPreheatTask[0].TaskId)
								sqlitedb.UpdateOrInsertreportdb(reportstatus)

								reportstart.ReportKpiLock.Lock()
								reportstart.ReportKpiStat.PreheatSuccNum++
								reportstart.ReportKpiLock.Unlock()

								str := fmt.Sprintf("taskid: [%d], result: [%d], err_code: [%d], err_msg: [%s], start_time: [%d], end_time[%d], tasktype: [inject], Url: [%s], dstaddr: [%s]",
									reportstatus.SubTaskId, reportstatus.Status, reportstatus.Err_code, reportstatus.Err_Msg, reportstatus.Start_time, reportstatus.End_time, reportstatus.Url, devip)
								logger.RecordLog(str)
							}
							return nil, code, err_msg
						default:
							logger.ErrorLog("data type is unknow")
						}
					}
					sqlitedb.UpdateOrInsertreportdb(reportstatus)
					return nil, code, err_msg
				}
				sqlitedb.UpdateOrInsertreportdb(reportstatus)
				str = fmt.Sprintf("taskid: [%d], result: [%d], err_code: [%d], err_msg: [%s], start_time: [%d], end_time[%d], tasktype: [inject], Url: [%s], dstaddr: [%s]",
					reportstatus.SubTaskId, reportstatus.Status, reportstatus.Err_code, reportstatus.Err_Msg, reportstatus.Start_time, reportstatus.End_time, reportstatus.Url, devip)
				logger.RecordLog(str)
				continue
			}
		}
	} else if resp.StatusCode == http.StatusFound { //如果响应码为302,根据相应换设备发送
		loction, err := resp.Location()
		if err != nil {
			logger.ErrorLog(err)
			return err, http.StatusInternalServerError, "over retrytimes"
		}
		tempUrl := loction.Host
		indx := strings.Index(tempUrl, ":")
		dstip := strings.TrimSpace(tempUrl[:indx])
		nodeStatus := httpClient.GetNodeStatus(dstip)
		if nodeStatus > 5 {
			subpreheattask.ReqPreheatTask[0].ForceInject = 1
			str := fmt.Sprintf("dstip:[%s] error, send Msg to original device:[%s]", dstip, devip)
			logger.ErrorLog(str)
			report, err := SendPreheatToNgx(subpreheattask, devip)
			return err, report.Err_code, report.Err_Msg
		}
		str := fmt.Sprintf("send Msg: [%s] to new device:[%s]", string(task), dstip)
		logger.InfoLog(str)
		report, err := SendPreheatToNgx(subpreheattask, dstip)
		return err, report.Err_code, report.Err_Msg
	} else {
		str := fmt.Sprintf("put preheattask fail, task: [%s], url: [%s], err_code: ", subpreheattask.ReqPreheatTask[0].TaskId, subpreheattask.ReqPreheatTask[0].ResUrl)
		logger.ErrorLog(str, resp.StatusCode)
		err_msg = "over retytimes"
		code = resp.StatusCode
		return errors.New(strconv.Itoa(resp.StatusCode)), code, err_msg
	}
	return nil, code, err_msg
}

//处理M3U8---备用
//func processM3u8(resp *http.Response, subpreheattask commonstruct.TempReqPreheatStruct, devip string) error {
//	var ngxReport commonstruct.RespPreheatStruct
//	var reportstatus commonstruct.ReportStatus
//	if resp.StatusCode == http.StatusOK { //如果响应码为200
//		body, err := ioutil.ReadAll(resp.Body)
//		PreheatGoroutines--
//		if err != nil {
//			str := fmt.Sprintf("put preheattask to device: [%s] fail, task: [%s], url: [%s], ",
//				devip, subpreheattask.ReqPreheatTask[0].TaskId, subpreheattask.ReqPreheatTask[0].ResUrl)
//			logger.ErrorLog(str, err)
//			resp.Body.Close()
//			return err
//		}
//		json.Unmarshal(body, &ngxReport)
//		resp.Body.Close()
//		var subTask []commonstruct.ReqPreheatStruct
//		reader := strings.NewReader(ngxReport.SubTaskUrls)
//		r := bufio.NewReader(reader)
//		for {
//			b, _, err := r.ReadLine()
//			if err == io.EOF {
//				str := fmt.Sprintf("put preheattask to device: [%s] success, task: [%s], url: [%s], RcvMsg: [%s]",
//					devip, subpreheattask.ReqPreheatTask[0].TaskId, subpreheattask.ReqPreheatTask[0].ResUrl, string(body))
//				logger.ErrorLog(str, err)
//				break
//			}
//			subpreheattask.ReqPreheatTask[0].ResUrl = string(b)
//			subpreheattask.ReqPreheatTask[0].UrlType = "2"
//			subpreheattask.ReqPreheatTask[0].Md5 = ""
//			subTask = append(subTask, subpreheattask.ReqPreheatTask[0])
//		}
//		start_time := time.Now().Unix()
//		err = SendM3U8toNgx(subTask, devip)
//		end_time := time.Now().Unix()
//		if err != nil {
//			reportstatus.SubTaskId,_ = strconv.Atoi(subpreheattask.ReqPreheatTask[0].TaskId)
//			reportstatus.Status = FALIED
//			reportstatus.Start_time = start_time
//			reportstatus.End_time = end_time
//			reportstatus.Level = subpreheattask.Level
//			reportstatus.Url = subpreheattask.ReqPreheatTask[0].ResUrl
//			//reportstatus.Err_code = http.StatusRequestTimeout
//			reportstatus.Err_Msg = err.Error()
//			//入库
//			sqlitedb.UpdateOrInsertreportdb(reportstatus)
//		} else{
//			reportstatus.SubTaskId,_ =  strconv.Atoi(subpreheattask.ReqPreheatTask[0].TaskId)
//			reportstatus.Status = SUCCESS
//			reportstatus.Start_time = start_time
//			reportstatus.End_time = end_time
//			reportstatus.Level = subpreheattask.Level
//			reportstatus.Url=subpreheattask.ReqPreheatTask[0].ResUrl
//			reportstatus.Err_code = http.StatusOK
//			//入库
//			sqlitedb.UpdateOrInsertreportdb(reportstatus)
//		}
//		return nil
//	} else {
//		str := fmt.Sprintf("put preheattask to device: [%s] fail, task: [%s], url: [%s], err_code: ", devip, subpreheattask.ReqPreheatTask[0].TaskId, subpreheattask.ReqPreheatTask[0].ResUrl)
//		logger.ErrorLog(str, resp.StatusCode)
//		PreheatGoroutines--
//		resp.Body.Close()
//		return errors.New(strconv.Itoa(resp.StatusCode))
//	}
//}
////发送M3U8并统计上报结果
//func SendM3U8toNgx(preheattask []commonstruct.ReqPreheatStruct, devip string) error {
//	var wg sync.WaitGroup
//	msg := make(map[string][]commonstruct.ReportStatus)
//	for i,_:=range preheattask{
//		wg.Add(1)
//		go func() {
//			defer wg.Done()
//			err := SendM3u8UrLToNgx(preheattask[i],devip)
//			var sreportstatus commonstruct.ReportStatus
//			if err != nil {
//				sreportstatus.SubTaskId,_ = strconv.Atoi(preheattask[i].TaskId)
//				sreportstatus.Status = Preheat_Failed
//				msg[preheattask[i].TaskId] = append(msg[preheattask[i].TaskId],sreportstatus)
//				str := fmt.Sprintf("put preheattask to device: [%s] fail, task: [%s], url: [%s], ", devip, preheattask[i].TaskId, preheattask[i].ResUrl)
//				logger.InfoLog(str,err)
//			}else {
//				sreportstatus.SubTaskId,_  = strconv.Atoi(preheattask[i].TaskId)
//				sreportstatus.Status = Completed
//				msg[preheattask[i].TaskId] = append(msg[preheattask[i].TaskId],sreportstatus)
//				str := fmt.Sprintf("put preheattask to device: [%s] success, task: [%s], url: [%s], ", devip, preheattask[i].TaskId, preheattask[i].ResUrl)
//				logger.InfoLog(str,err)
//			}
//		}()
//	}
//	wg.Wait()
//	for i,_:=range msg[preheattask[0].TaskId]{
//		if msg[preheattask[i].TaskId][i].Status == Preheat_Failed{
//			return  errors.New("over retrytimes")
//		}
//	}
//	return  nil
//}
////发送M3U8解析结果
//func SendM3u8UrLToNgx(preheattask commonstruct.ReqPreheatStruct, devip string) (error) {
//	var tmpTask []commonstruct.ReqPreheatStruct
//	tmpTask = append(tmpTask, preheattask)
//	task, _ := json.Marshal(tmpTask)
//	for j := 0; j < preheatRetryTimes+1; j++ {
//		if j == preheatRetryTimes {
//			return errors.New("fail")
//		}
//		times := j * preheatRetryTime
//		time.Sleep(time.Duration(times) * time.Minute)
//		PreheatGoroutines++
//		resp, err := httpClient.SendPreheatTask(devip, task)
//		if err != nil { //如果有异常
//			str := fmt.Sprintf("put preheattask(m3u8) to device: [%s] fail, task: [%s], url: [%s], faildtimes[%d] ",
//				devip, preheattask.TaskId, preheattask.ResUrl, j+1)
//			logger.ErrorLog(str, err)
//			PreheatGoroutines--
//			continue
//		}
//		var ngxReport commonstruct.RespPreheatStruct
//		if resp.StatusCode == http.StatusOK { //如果响应码为200
//			reader := bufio.NewReader(resp.Body)
//			var tmpStr string
//			for {
//				line, err := reader.ReadByte()
//				if err != nil {
//					if err == io.EOF {
//						PreheatGoroutines--
//						str := fmt.Sprintf("put preheattask(m3u8) to device: [%s] success, task: [%s], url: [%s]",
//							devip, preheattask.TaskId, preheattask.ResUrl)
//						logger.InfoLog(str)
//						resp.Body.Close()
//						return nil
//					}
//					str := fmt.Sprintf("put preheattask(m3u8) to device: [%s] fail, task: [%s], url: [%s],  faildtimes[%d]",
//						devip, preheattask.TaskId, preheattask.ResUrl, j+1)
//					logger.ErrorLog(str, err)
//					resp.Body.Close()
//					PreheatGoroutines--
//					break
//				}
//				if line =='{'{
//					tmpStr = tmpStr + string(line)
//					continue
//				}
//				if line !='}'{
//					tmpStr = tmpStr + string(line)
//					continue
//				}
//				if line =='}' {
//					tmpStr = tmpStr + string(line)
//					PreheatGoroutines--
//					json.Unmarshal([]byte(tmpStr), &ngxReport)
//					tmpStr = ""
//					if ngxReport.Result == Preheat_Failed {
//						resp.Body.Close()
//						break
//					}
//				}
//			}
//		} else if resp.StatusCode == http.StatusFound {
//			loction, _ := resp.Location()
//			tempUrl := loction.Opaque
//			indx := strings.Index(tempUrl, "/")
//			tempUrl = strings.TrimSpace(tempUrl[indx+2:])
//			indx = strings.Index(tempUrl, "/")
//			url := strings.TrimSpace(tempUrl[:indx])
//			PreheatGoroutines--
//			resp.Body.Close()
//			str := fmt.Sprintf("send Msg: [%s] to new device:[%s]", string(task), url)
//			logger.InfoLog(str)
//			err := SendM3u8UrLToNgx(preheattask, url)
//			return err
//		} else {
//			str := fmt.Sprintf("put preheattask(m3u8) to device: [%s] fail, task: [%s], url: [%s], err_code: [%d], failedtimes: [%d]",
//				devip, preheattask.TaskId, preheattask.ResUrl, resp.StatusCode, j+1)
//			logger.ErrorLog(str)
//			PreheatGoroutines--
//			resp.Body.Close()
//			continue
//		}
//	}
//	return nil
//}
//MD5查询
func QueryTask(queryTask commonstruct.TempReqPreheatStruct, respNgx commonstruct.RespPreheatStruct, devip string) error {
	if queryTask.ReqPreheatTask[0].Md5 == "" {
		logger.InfoLog("don't need query MD5")
		return nil
	}
	time.Sleep(200 * time.Millisecond)
	var reqQuery []commonstruct.Md5QueryStruct
	reqQuery = make([]commonstruct.Md5QueryStruct, 1)
	reqQuery[0].UrlType = queryTask.ReqPreheatTask[0].UrlType
	reqQuery[0].TaskId = queryTask.ReqPreheatTask[0].TaskId
	reqQuery[0].ResUrl = queryTask.ReqPreheatTask[0].ResUrl
	reqQuery[0].ResKey = respNgx.ResKey
	reqQuery[0].Hot = queryTask.ReqPreheatTask[0].Hot
	reqQuery[0].Md5 = queryTask.ReqPreheatTask[0].Md5

	task, err := json.Marshal(reqQuery)
	if err != nil {
		logger.ErrorLog(err)
		return err
	}
	//发送请求
	resp, err := httpClient.SendQueryTask(devip, task)
	if err != nil {
		str := fmt.Sprintf("put querytask to device: [%s] fail, task: [%s]", devip, string(task))
		logger.ErrorLog(str, err)
		return err
	}
	defer resp.Body.Close()
	var ngxReport commonstruct.RespMd5QueryStruct
	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			str := fmt.Sprintf("put querytask to device: [%s] fail, task: [%s]", devip, string(task))
			logger.ErrorLog(str, err)
			return err
		}
		str := fmt.Sprintf("get queryResp from Ngx, RcvMsg:%s", string(body))
		logger.InfoLog(str)
		err = json.Unmarshal(body, &ngxReport)
		if err != nil {
			logger.ErrorLog("json parse failed", err)
			return err
		}
		if ngxReport.Result != 2 || ngxReport.Md5Check != 0 {
			logger.ErrorLog("MD5 checkout failed, RcvMsg: ", string(body))
			return errors.New("MD5 checkout failed")
		} else {
			logger.InfoLog("MD5 checkout success, RcvMsg: ", string(body))
			return nil
		}
	} else {
		logger.InfoLog("MD5 checkout failed, errcode: ", resp.StatusCode)
		return errors.New("MD5 checkout failed")
	}
	return nil
}
