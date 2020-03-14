package gettask

import (
	"net/http"
	"common/logger"
	"time"
	"io/ioutil"
	"errors"
	"encoding/json"
	"common/sqlitedb"
	"common/commonstruct"
	"common/conf"
	"strings"
	"crypto/tls"
	"common/httpClient"
	"preheatmcserv/reportstart"
)
func GetTaskfromHTTP()(error){
	//devInfo,err := sqlitedb.Querydevicedb()
	//if err != nil{
	//	return err
	//}
	defer func() {
		if errs := recover(); errs != nil {
			logger.ErrorLog("CMSAgent runtime err, panic: ", errs)
		}
		return
	}()
	url := conf.Config["srcaddr"] + conf.Config["srcinterface"] + conf.Config["vip"] //拼接拉取任务URL
	//获取鉴权部分
	values, err := httpClient.GetAuthorization("gettask")
	if err != nil { //如果有异常
		logger.ErrorLog("Get Gettask Authorization failed, ", err)
		return err
	}
	tr := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}
	client := http.Client{
		Transport: tr,
		Timeout:60*time.Second,
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil{
		logger.ErrorLog(err)
		return err
	}
	request.Header.Add("Authorization",values)
	resp, err := client.Do(request)
	if err != nil {                        //如果有异常
		logger.ErrorLog("get task failed, ",err)
		return	err
	}
	defer resp.Body.Close() //go的特殊语法，main函数执行结束前会执行resp.Body.Close()
	if resp.StatusCode == http.StatusOK { //如果响应码为200
		body, err := ioutil.ReadAll(resp.Body) //把响应的body读出
		if err != nil { //如果有异常
			logger.ErrorLog("read response body faield, ", err)
			return err
		}
		var getTaskInfo commonstruct.RespGetTaskList

		//解析拉取的任务
		err = json.Unmarshal(body,&getTaskInfo)
		if err != nil{
			logger.ErrorLog("decode task fail, ",err)
			return err
		}
		//logger.InfoLog("get task from sam, task:, ", string(body))
		if getTaskInfo.Error_msg != "success."{
			return errors.New(getTaskInfo.Error_msg)
		}
		var refreshTask commonstruct.DbInsertStruct
		var preheatTask commonstruct.DbInsertStruct
		for i := 0; i<getTaskInfo.TaskNum; i++{
			if getTaskInfo.Tasks[i].TaskType == "refresh"{
				//刷新
				dbTask := commonstruct.GetInsertSqliteDataStruct(getTaskInfo.Tasks[i])
				//插入刷新数据库
				refreshTask.Num++
				refreshTask.Tasks = append(refreshTask.Tasks, dbTask)
			}
			if getTaskInfo.Tasks[i].TaskType == "inject"{
				length := len(getTaskInfo.Tasks[i].ContentUrl)
				//解析预热url类型
				tempStr := strings.TrimSpace(getTaskInfo.Tasks[i].ContentUrl[length-4:])
				//预热
				dbTask := commonstruct.GetInsertSqliteDataStruct(getTaskInfo.Tasks[i])
				//插入预热数据库
				if tempStr == "m3u8"{
					dbTask.UrlType = "5"
					continue
				}else {
					dbTask.UrlType = "2"
				}
				preheatTask.Num++
				preheatTask.Tasks = append(preheatTask.Tasks, dbTask)
			}
		}
		if preheatTask.Num != 0{
			err := sqlitedb.Insertpreheatdb(preheatTask)
			if err != nil{
				return err
			}
		}
		if refreshTask.Num != 0{
			err = sqlitedb.Insertrefreshdb(refreshTask)
			if err != nil{
				return err
			}
		}
		reportstart.ReportKpiStat.TotalTaskNum += uint64(getTaskInfo.TaskNum)
		reportstart.ReportKpiStat.PreheatTaskNum += uint64(preheatTask.Num)
		reportstart.ReportKpiStat.RefreshTaskNum += uint64(refreshTask.Num)
		return nil
	}
	//body, err := ioutil.ReadAll(resp.Body) //把响应的body读出
	logger.ErrorLog("get task fail, statusCode: ",resp.StatusCode)
	return errors.New(resp.Status)
}


