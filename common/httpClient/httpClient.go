package httpClient

import (
	"common/conf"
	"common/logger"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)
const(
	Load_Status = "load_status"
	Refresh = "refresh"
	Preheat = "preheat"
	Query = "query"
	MaxStattus = 10
)
var PreheatClient http.Client
var GetNodeClient http.Client
var RefreshClient http.Client
var QueryClient http.Client
//防止302内部自动跳转
func noRedirect(req *http.Request, via []*http.Request) error {
	return errors.New("302")
}
func init(){
	tr := &http.Transport{
		DisableKeepAlives: true,
	}
	PreheatClient = http.Client{
		CheckRedirect:noRedirect,
		Timeout:1*time.Minute,
		Transport:tr,
	}
	GetNodeClient = http.Client{
		Timeout:1*time.Minute,
		Transport:tr,
	}
	RefreshClient = http.Client{
		Timeout:1*time.Minute,
		Transport:tr,
	}
	QueryClient = http.Client{
		CheckRedirect:noRedirect,
		Timeout:1*time.Minute,
		Transport:tr,
	}
}

func GetRequestHead(request *http.Request, flag string, task string){
	if flag == Load_Status{
		request.Proto="HTTP/1.1"
		request.Header.Add("User-Agent","HW-Cache/2.2.00")
		request.Header.Add("Host","127.0.0.1")
		request.Header.Add("Connection","close")
	}
	if flag == Refresh || flag == Preheat || flag == Query{
		request.Proto="HTTP/1.1"
		request.Header.Add("User-Agent","HW-HCS-INNER/1.0.00")
		request.Header.Add("Host","127.0.0.1")
		//request.Header.Add("Accept","*/*")
		request.Header.Add("Content-Type","application/json")
		request.Header.Add("Connection","close")
		request.Header.Add("Content-Length", strconv.Itoa(len([]byte(task))))
		request.Header.Add("Cms-Agent", "1")
	}
}

func GetNodeStatus(dstip string) int {
	var nodeStatus	int
	//拼接获取负载状态的url
	dsturl := "http://" + dstip + conf.Config["getnodestatus"]
	//建立请求
	request, err := http.NewRequest("GET", dsturl, nil)
	if err != nil{
		logger.ErrorLog(err)
		return MaxStattus
	}
	//填充请求头
	GetRequestHead(request, Load_Status,"")
	//发送请求
	resp, err := GetNodeClient.Do(request)
	if err != nil{
		logger.ErrorLog("get hcs nodeStatus fail, devip: ",dstip, err)
		return MaxStattus
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK{
		body, err := ioutil.ReadAll(resp.Body) //把响应的body读出
		if err != nil{
			logger.ErrorLog(err)
			return MaxStattus
		}

		//str := fmt.Sprintf("get hcs nodeStatus, devip: [%s], nodeStatus: [%s]", dstip, string(body))
		//logger.InfoLog(str)
		s := strings.TrimSpace(string(body))
		if strings.Contains(s, ":"){
			firstIndx := strings.Index(s, ":")
			nodeStatus,err = strconv.Atoi(strings.TrimSpace(s[firstIndx+2:]))
			if err != nil{
				logger.ErrorLog(err)
				nodeStatus = MaxStattus
			}
		}else {
			return MaxStattus
		}

	}else {
		str := fmt.Sprintf("get hcs nodeStatus fail, devip: [%s], errcode: [%d]", dstip, resp.StatusCode)
		logger.ErrorLog(str)
		return MaxStattus
	}
	return nodeStatus
}

//发送刷新任务
func SendRefreshTask(dstip string, task []byte) (*http.Response, error){
	dsturl := "http://" + dstip + conf.Config["refreshdstport"]
	request, err := http.NewRequest("POST", dsturl, strings.NewReader(string(task)))
	if err != nil{
		logger.ErrorLog("make refreshtask request err, ",err)
	}
	//填充请求头
	GetRequestHead(request, Refresh, string(task))
	logger.ErrorLog("putrefreshtask to ngx, task: ",string(task)," destip: ",dstip)
	resp,err := RefreshClient.Do(request)
	return resp,err
}
//发送预热任务
func SendPreheatTask(dstip string, task []byte) (*http.Response, error){
	dsturl := "http://" + dstip + conf.Config["preheatdstport"]
	request, err := http.NewRequest("POST", dsturl, strings.NewReader(string(task)))
	if err != nil{
		logger.ErrorLog("make preheattask request err, ",err)
	}
	//请求头
	GetRequestHead(request, Preheat, string(task))
	logger.ErrorLog("putpreheattask to ngx, task: ",string(task)," destip: ",dstip)
	resp,err := PreheatClient.Do(request)
	if err != nil {
		logger.ErrorLog("putpreheattask to ngx error, ",err)
		if resp != nil{
			if resp.StatusCode == 302 {
				err = nil
			}
		}
	}
	return resp,err
}
//发送查询
func SendQueryTask(dstip string, task []byte) (*http.Response, error){
	dsturl := "http://" + dstip + conf.Config["querydstport"]
	request, err := http.NewRequest("POST", dsturl, strings.NewReader(string(task)))
	if err != nil {
		logger.ErrorLog(err)
	}
	//请求头
	GetRequestHead(request, Query, string(task))
	logger.InfoLog("putquerytask to ngx, task: ",string(task)," destip: ",dstip)
	resp,err := QueryClient.Do(request)
	return resp,err
}

//与服务端之间进行加密
func GetAuthorization(flag string)(string,error){
	time_Now := strings.ToUpper(strconv.FormatInt(time.Now().Unix(),16))
	var randNum = []rune("1234567890ABCDEF")
	tmpNum := make([]rune,16)
	for i := range tmpNum{
		tmpNum[i] = randNum[rand.Intn(len(randNum))]
	}
	randMath := string(tmpNum)
	var key string
	if flag == "gettask"{
		key = conf.Config["srcinterface"] + conf.Config["vip"]  + time_Now
	}
	if flag == "report"{
		key = "/task/report_status"  + time_Now
	}
	realKey := hmac.New(sha256.New, []byte("cmnet"))
	_, err := realKey.Write([]byte(randMath))
	if err != nil {
		logger.ErrorLog("mac sha256 auth write error for realKey, ",err)
		return "",err
	}
	realKeyStr := hex.EncodeToString(realKey.Sum(nil))
	calculateAuth := hmac.New(sha256.New, []byte(realKeyStr))
	_, err = calculateAuth.Write([]byte(key))
	if err != nil {
		logger.ErrorLog("mac sha256 auth write error, ",err)
		return "",err
	}
	//全部转换成大写
	calculateMAC := strings.ToUpper(hex.EncodeToString(calculateAuth.Sum(nil)))
	value := randMath + "|" + time_Now + "|" + calculateMAC
	return value,nil
}
