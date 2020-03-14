package parsefile

import (
	"bufio"
	"common/commonstruct"
	"common/logger"
	"flag"
	"fmt"
	"github.com/larspensjo/config"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	pubPath      = "/home/icache/common/config/pub.conf"
	netWorikPath = "/home/icache/hcs/hcs/conf/network.ini"
	zkConfigPath = "/home/icache/hcs/hcs/conf/dev_platform.xml"
	LocalIpPath  = "/home/icache/hcs/hcs/conf/config.xml"
)

var (
	netWorkFile = flag.String("configurefile", netWorikPath, "General configuration file")
)
var TempGroupInfo = make(map[string]string)
var TemGroupIdList = make(map[string]string)
var TemBusinessIpInfo = make(map[string]string)

func Parsenetwork() (commonstruct.Groupinfo, error) {
	var localGroupInfo commonstruct.Groupinfo
	//读取本地ip
	localip, err := GetLocalIp()
	if err != nil {
		logger.ErrorLog("get localip failed, ", err)
		return localGroupInfo, err
	}
	//redisdb := redis.NewFailoverClient(&redis.FailoverOptions{
	//	MasterName:    "master",
	//	SentinelAddrs: []string{"127.0.0.1:20355","127.0.0.1:20354","127.0.0.1:20353"},
	//})
	//redisdb.Ping()
	//res,err:=redisdb.Get("Network").Result()
	//if err!=nil{
	//	logger.ErrorLog("redis connect fail",err)
	//	return localGroupInfo,err
	//}
	//temp:=strings.NewReader(res)
	//r:=bufio.NewReader(temp)
	//runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	//set config file std

	cfg, err := config.ReadDefault(*netWorkFile)
	if err != nil {
		logger.ErrorLog("Fail to find", *netWorkFile, err)
		return localGroupInfo, err
	}
	logger.InfoLog("network.ini parse start")
	if cfg.HasSection("device") {
		section, err := cfg.SectionOptions("device")
		if err == nil {
			for _, v := range section {
				options, err := cfg.String("device", v)
				if err == nil {
					if strings.Contains(v, "hcs") && !strings.Contains(v, "hcscount") {
						//fmt.Println(string(b))
						TempGroupInfo[v] = options
					}
				} else {
					logger.ErrorLog(err)
					return localGroupInfo, err
				}
			}
		} else {
			logger.ErrorLog(err)
			return localGroupInfo, err
		}
	}
	logger.InfoLog("get hcsinfo success,", TempGroupInfo)
	for _, info := range TempGroupInfo {
		index := strings.Index(info, ",")
		deviceId := strings.TrimSpace(info[:index])
		str := "groupInfo" + deviceId
		if cfg.HasSection(str) {
			section, err := cfg.SectionOptions(str)
			if err == nil {
				for _, v := range section {
					options, err := cfg.String(str, v)
					if err == nil {
						TemGroupIdList[str] = options
					} else {
						logger.ErrorLog(err)
						return localGroupInfo, err
					}
				}
			} else {
				logger.ErrorLog(err)
				return localGroupInfo, err
			}
		}
		str1 := "portalInfo" + deviceId
		if cfg.HasSection(str1) {
			section, err := cfg.SectionOptions(str1)
			if err == nil {
				for _, v := range section {
					options, err := cfg.String(str1, v)
					if err == nil {
						if strings.Contains(v, "portalData1") {
							firstindex := strings.Index(options, "/")
							str := strings.TrimSpace(options[firstindex+2:])
							lastindex := strings.Index(str, ":")
							TemBusinessIpInfo[str1] = strings.TrimSpace(str[:lastindex])
						}
					} else {
						logger.ErrorLog(err)
						return localGroupInfo, err
					}
				}
			} else {
				logger.ErrorLog(err)
				return localGroupInfo, err
			}
		}
	}
	logger.InfoLog("get groupInfo success,", TemGroupIdList)
	logger.InfoLog("get BusinessIp success,", TemBusinessIpInfo)
	//logger.InfoLog("get groupid info:", TemGroupIdList)
	//根据自己设备ip填充localGroupInfo
	//var Dev commonstruct.DevInfo
	var localDevId string
	var popid int
	for i := range TempGroupInfo {
		if strings.Contains(TempGroupInfo[i], localip) {
			indx := strings.Index(TempGroupInfo[i], ",")
			localDevId = strings.TrimSpace(TempGroupInfo[i][:indx])
			lastindx := strings.LastIndex(TempGroupInfo[i], ",")
			popid, err = strconv.Atoi(strings.TrimSpace(TempGroupInfo[i][lastindx+1:]))
			if err != nil {
				logger.ErrorLog("pop id type is err", err)
			}
			break
		}
	}

	groupid := TemGroupIdList["groupInfo"+localDevId]
	var tempdevid []int
	for i := range TemGroupIdList {
		if groupid == TemGroupIdList[i] {
			devid, err := strconv.Atoi(strings.TrimSpace(i[len("groupInfo"):]))
			if err != nil {
				logger.ErrorLog("device id type is err", err)
			}
			tempdevid = append(tempdevid, devid)
		}
	}
	//logger.InfoLog("localGroup deviceId: ", tempdevid)
	localGroupInfo.GroupiId, err = strconv.Atoi(groupid)
	if err != nil {
		logger.ErrorLog("get GroupId error: ", err)
		return localGroupInfo, err
	}
	localGroupInfo.PopId = popid
	for i := range TempGroupInfo {
		firstindx := strings.Index(TempGroupInfo[i], ",")
		tempDevId := strings.TrimSpace(TempGroupInfo[i][:firstindx])
		for j := range tempdevid {
			if tempDevId == strconv.Itoa(tempdevid[j]) {
				devip := TemBusinessIpInfo["portalInfo"+tempDevId]
				localGroupInfo.DevInfoList = append(localGroupInfo.DevInfoList, commonstruct.DevInfo{tempdevid[j], devip, 0})
			}
		}
	}
	logger.ErrorLog("localGroup info: ", localGroupInfo)
	return localGroupInfo, nil
}

//func ParsePub()(string, error){
//	var businessIp string
//	var businessvIp4 string
//	var businessvIp6 string
//	f, err := os.Open(pubPath)
//	defer f.Close()
//	//异常处理 以及确保函数结尾关闭文件流
//	if err != nil {
//		str := fmt.Sprintf("open [%s] fail, ", pubPath)
//		logger.ErrorLog(str, err)
//		return businessvIp4,err
//	}
//	//创建一个输出流向该文件的缓冲流*Reader
//	r := bufio.NewReader(f)
//	for {
//		b, _, err := r.ReadLine()
//		if err != nil {
//			if err == io.EOF {
//				break
//			}
//			str := fmt.Sprintf("read [%s] fail, ", pubPath)
//			logger.ErrorLog(str, err)
//			return businessvIp4,err
//		}
//		s := strings.TrimSpace(string(b))
//		if strings.Contains(s, "<business_ip>"){
//			firstIndx := strings.Index(s, ">")
//			lastIndx := strings.LastIndex(s, "<")
//			businessIp = strings.TrimSpace(s[firstIndx+1:lastIndx])
//		}
//		if strings.Contains(s, "<business_vips>"){
//			firstIndx := strings.Index(s, ">")
//			lastIndx := strings.LastIndex(s, "<")
//			businessvIp4 = strings.TrimSpace(s[firstIndx+1:lastIndx])
//		}
//		if strings.Contains(s, "<ipv6_business_vips>"){
//			firstIndx := strings.Index(s, ">")
//			lastIndx := strings.LastIndex(s, "<")
//			businessvIp6 = strings.TrimSpace(s[firstIndx+1:lastIndx])
//			break
//		}
//	}
//	if businessvIp4 !="" && businessvIp4 != "0:;1:" && !strings.Contains(businessvIp4,"127.0.0.1"){
//		logger.RecordLog("get businessvip4 success, ip: ",businessvIp4)
//		indx := strings.Index(businessvIp4,":")
//		lastindx := strings.Index(businessvIp4,";")
//		return strings.TrimSpace(businessvIp4[indx+1:lastindx]),nil
//	}
//	if businessvIp6 !="" && businessvIp6 != "0:;1:" && !strings.Contains(businessvIp6,"127.0.0.1"){
//		logger.RecordLog("get businessvip6 success, ip: ",businessvIp6)
//		indx := strings.Index(businessvIp6,":")
//		lastindx := strings.Index(businessvIp6,";")
//		return strings.TrimSpace(businessvIp6[indx+1:lastindx]),nil
//	}
//	if businessIp !="" && strings.TrimSpace(businessIp[:1]) != "0" && businessIp != "127.0.0.1"{
//		logger.RecordLog("get businessip success, ip: ",businessIp)
//		return businessIp,nil
//	}
//	logger.ErrorLog("get vip failed, vip is nil")
//	return "",errors.New("failed")
//}
func ParseXml() (error, []string) {
	var recordIp []string
	file, err := os.Open(zkConfigPath)
	if err != nil {
		str := fmt.Sprintf("open [%s] fail, ", zkConfigPath)
		logger.ErrorLog(str, err)
		return err, recordIp
	}
	defer file.Close()
	//创建一个输出流向该文件的缓冲流*Reader
	r := bufio.NewReader(file)
	for {
		b, _, err := r.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			str := fmt.Sprintf("read [%s] fail, ", zkConfigPath)
			logger.ErrorLog(str, err)
			return err, recordIp
		}
		s := strings.TrimSpace(string(b))
		if strings.Contains(s, "<recordIp>") {
			firstIndx := strings.Index(s, ">")
			lastIndx := strings.LastIndex(s, "<")
			recordIp = append(recordIp, strings.TrimSpace(s[firstIndx+1:lastIndx]))
		}
	}
	logger.InfoLog("zkServers address: ", recordIp)
	return nil, recordIp
}
func GetLocalIp() (string, error) {
	var localip string
	file, err := os.Open(LocalIpPath)
	//异常处理 以及确保函数结尾关闭文件流
	if err != nil {
		str := fmt.Sprintf("open [%s] fail, ", LocalIpPath)
		logger.ErrorLog(str, err)
		return localip, err
	}
	defer file.Close()
	//创建一个输出流向该文件的缓冲流*Reader
	rd := bufio.NewReader(file)
	for {
		b, _, err := rd.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			str := fmt.Sprintf("read [%s] fail, ", LocalIpPath)
			logger.ErrorLog(str, err)
			return localip, err
		}
		s := strings.TrimSpace(string(b))
		if strings.Contains(s, "<ipAddress>") {
			firstIndx := strings.Index(s, ">")
			lastIndx := strings.LastIndex(s, "<")
			localip = strings.TrimSpace(s[firstIndx+1 : lastIndx])
		}
	}
	logger.InfoLog("get localip success, ip: ", localip)
	return localip, nil
}
func GetBusinessIp() (string, error) {
	var localip string
	file, err := os.Open(pubPath)
	//异常处理 以及确保函数结尾关闭文件流
	if err != nil {
		str := fmt.Sprintf("open [%s] fail, ", pubPath)
		logger.ErrorLog(str, err)
		return localip, err
	}
	defer file.Close()
	//创建一个输出流向该文件的缓冲流*Reader
	rd := bufio.NewReader(file)
	for {
		b, _, err := rd.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			str := fmt.Sprintf("read [%s] fail, ", pubPath)
			logger.ErrorLog(str, err)
			return localip, err
		}
		s := strings.TrimSpace(string(b))
		if strings.Contains(s, "<business_ip>") {
			firstIndx := strings.Index(s, ">")
			lastIndx := strings.LastIndex(s, "<")
			localip = strings.TrimSpace(s[firstIndx+1 : lastIndx])
		}
	}
	logger.InfoLog("get localip success, ip: ", localip)
	return localip, nil
}
func GetlocalId() (int, string) {
	localGroupInfo, err := Parsenetwork()
	if err != nil {
		logger.ErrorLog(err)
	}
	localip, err := GetBusinessIp()
	if err != nil {
		logger.ErrorLog(err)
	}
	localId := 0
	for i := range localGroupInfo.DevInfoList {
		if localip == localGroupInfo.DevInfoList[i].DevIp {
			localId = localGroupInfo.DevInfoList[i].DevId
		}
	}
	return localId, localip
}
