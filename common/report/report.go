package report

import (
	"net/http"
	"encoding/json"
	"common/commonstruct"
	"time"
	"strings"
	"io/ioutil"
	"errors"
	"strconv"
	"common/logger"
	"common/conf"
	"common/httpClient"
	"crypto/tls"
)
var ReSendNumList = make(map[string]int)//失败重发次数
//上报状态请求结构体

func StatusReport(reporttask commonstruct.ReportList) error {
	defer func() {
		if errs := recover(); errs != nil {
			logger.ErrorLog("CMSAgent runtime err, panic: ", errs)
		}
		return
	}()
	url := conf.Config["srcaddr"] + "/task/report_status"
	//获取鉴权部分
	values, err := httpClient.GetAuthorization("report")
	if err != nil { //如果有异常
		logger.ErrorLog("Get report Authorization failed, ", err)
		return err
	}
	tr := &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	}
	js, err := json.Marshal(reporttask)
	if err != nil{
		logger.ErrorLog(err)
		return err
	}
	client := http.Client{
		Transport: tr,
		Timeout:60*time.Second,
	}
	logger.InfoLog("report msg: ", string(js))
	request, err := http.NewRequest("POST", url,  strings.NewReader(string(js)))
	if err != nil{
		logger.ErrorLog(err)
		return err
	}
	request.Header.Add("Authorization",values)
	request.Header.Add("Content-Type","application/json;charset=UTF-8")
	resp, err := client.Do(request)
	//错误日志处理
	if err != nil { //如果有异常
		logger.ErrorLog("report fail, ", err)
		return err
	}
	defer resp.Body.Close() //go的特殊语法，main函数执行结束前会执行resp.Body.Close()
	//var TaskInfo = make(map[string]string)
	if resp.StatusCode == http.StatusOK { //如果响应码为200
		body, err := ioutil.ReadAll(resp.Body) //把响应的body读出
		if err != nil { //如果有异常
			logger.ErrorLog("read report resp fail, ", err)
			return err
		}
		if strings.Contains(string(body),"success"){
			logger.InfoLog("report result: ", string(body))
			return nil
		}else {
			logger.ErrorLog("report failed: ", string(body))
			return errors.New(string(body))
		}
	}
	return errors.New("statusCode: " + strconv.Itoa(resp.StatusCode))
}
