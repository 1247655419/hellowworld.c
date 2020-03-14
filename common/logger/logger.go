package logger

/**
基于log4g的封装
*/
import (
	"fmt"
	"lib/log4g"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"net/http"
)

//var ConfPath string = "/home/icache/cms/conf/"
var ConfPath string = "/home/icache/hcs/hcs/conf/"

func Goid() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil{
		fmt.Println(err)
	}
	return id
}

func GetPosInfo() (file string, line int, funcname string) {
	funcName, file, line, ok := runtime.Caller(2)
	if(ok){
		funcname = runtime.FuncForPC(funcName).Name()
		file = path.Base(file)
		funcname = path.Base(funcname)
	}

	return file, line, funcname
}
func InitLog(v...interface{}) {
	//id := Goid()
	file, line, funcname := GetPosInfo()
	log4g.GetLogger("init").Error(file, ":", line, "|", funcname, "|", v)
}
func ErrorLog(v...interface{}) {
	//id := Goid()
	file, line, funcname := GetPosInfo()
	log4g.GetLogger("service").Error(file, ":", line, "|", funcname, "|", v)
}

func InfoLog(v...interface{}) {
	//id := Goid()
	file, line, funcname := GetPosInfo()
	log4g.GetLogger("service").Info(file, ":", line, "|", funcname, "|", v)
}

func DebugLog(v...interface{}) {
	id := Goid()
	file, line, funcname := GetPosInfo()
	log4g.GetLogger("service").Debug(id, "|", file, ":", line, "|", funcname, "|", v)
}

func RecordLog(v...interface{}) {
	log4g.GetLogger("record").Fatal(v...)
}

/*
 conffile: 配置文件名，eg： log4g-server.properties
*/

func init() {
	err := log4g.ConfigF(ConfPath + "log4g-server.properties")
	if err != nil {
		fmt.Println("Init logger falied, err:" + err.Error())
		os.Exit(-1)
	}else{
		fmt.Println("Init logger ok" )
	}
	InfoLog("Init logger ok")
}

func LoggerShutDown() {
	log4g.Shutdown()
}

func SetLoggerLevel(loggerName string, level string) bool {
	switch level {
	case "FATAL":
		log4g.SetLogLevel(loggerName, log4g.FATAL)
	case "ERROR":
		log4g.SetLogLevel(loggerName, log4g.ERROR)
	case "DEBUG":
		log4g.SetLogLevel(loggerName, log4g.DEBUG)
	case "INFO":
		log4g.SetLogLevel(loggerName, log4g.INFO)
	case "TRACE":
		log4g.SetLogLevel(loggerName, log4g.TRACE)
	default:
		return false
	}

	return true
}

func StartEasyServer() {
	http.HandleFunc("/loglevel", setLogLevel)

	err := http.ListenAndServe("127.0.0.1:20078", nil)
	if err != nil {
		fmt.Println("http binding same port error:", err.Error())
		os.Exit(-1)
	}
}

func setLogLevel(rw http.ResponseWriter, req *http.Request) {
	DebugLog("setLogLevel url:", req.URL.Path)

	level := req.URL.Query().Get("Level")

	ret := SetLoggerLevel("service", level)
	SetLoggerLevel("record", level)


	if ret != true {
		rw.WriteHeader(http.StatusOK)
		_, err := rw.Write([]byte("not support the level, support level: INFO, DEBUG, ERROR, FATAL, TRACE\n"))
		if err != nil {
			ErrorLog("set log level error:", err.Error())
		}
		return
	}

	rw.WriteHeader(http.StatusOK)
	_, err := rw.Write([]byte("Set the logger success:" + level + "\n"))
	if err != nil {
		ErrorLog("set log level of resposne to client error:", err.Error())
	}
}
