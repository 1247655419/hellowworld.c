package sqlitedb

import (
	"common/commonstruct"
	"common/conf"
	"common/logger"
	"common/parsefile"
	"common/zkelectmaster"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-concurrentMap-master"
	_ "github.com/mattn/go-sqlite3"
	"strconv"
	"sync"
	"time"
)

var RefreshDB *sql.DB
var PreheatDB *sql.DB

//var ReprotDB *sql.DB
//var DeviceDB *sql.DB
var refreshlock sync.Mutex
var preheatlock sync.Mutex
var reportlock sync.Mutex
var Devicelock sync.Mutex
var TaskQueueLock sync.Mutex
var DeviceInfo commonstruct.Groupinfo
var WaitPreheatTask = concurrent.NewConcurrentMap()
var ReportTask = concurrent.NewConcurrentMap()

var maxReportTaskNum int

//初始化
func Init() error {
	var err error
	maxReportTaskNum, err = strconv.Atoi(conf.Config["maxreport"])
	if err != nil {
		maxReportTaskNum = 1000
	}
	//创建刷新表
	RefreshDB, err = sql.Open("sqlite3", "/home/icache/hcs/hcs/db/refresh.db")
	//defer RefreshDB1.Close()
	if err != nil {
		logger.ErrorLog("dbrefresh init fail, ", err)
		return err
	}
	rows, err := RefreshDB.Query("select name from sqlite_master where type='table' order by name")
	if err != nil {
		logger.ErrorLog("query refresh init fail, ", err)
		return err
	}
	if rows.Next() {
		rows.Close()
		_, err = RefreshDB.Exec("DROP INDEX inx_TaskId;")
		if err != nil {
			logger.ErrorLog("delete refreshdb index taskid error, ", err)
			return err
		}
		_, err = RefreshDB.Exec("DROP INDEX inx_Status;")
		if err != nil {
			logger.ErrorLog("delete refreshdb index status error, ", err)
			return err
		}
	} else {
		rows.Close()
	}
	refreshinfo := `
    CREATE TABLE IF NOT EXISTS refreshinfo(
        Uid INTEGER PRIMARY KEY ,
        TaskId VARCHAR(64) NOT NULL,
        ResUrl VARCHAR(200) NOT NULL,
 		AccessUrl VARCHAR(200) NULL,
 		Md5 VARCHAR(20) NULL,
        UrlType VARCHAR(2) NOT NULL,
		ResKey VARCHAR(50) NULL,
		ScopeType  VARCHAR(2) NULL,
		FleeSwitch VARCHAR(2) NULL,
		FleeIp VARCHAR(20) NULL,
		BackSourceIp VARCHAR(20) NULL,
		Hot VARCHAR(2) NULL,
		RangeBegin VARCHAR(5) NULL,
		RangeEnd VARCHAR(5) NULL,
		Levels NUMBER NOT NULL, 
		Status NUMBER	NOT NULL,
		Retrytimes NUMBER NULL,
		Times NUMBER  NOT NULL 
    );`
	_, err = RefreshDB.Exec(refreshinfo)
	if err != nil {
		logger.ErrorLog("dbrefresh create fail, ", err)
		return err
	}
	_, err = RefreshDB.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		logger.ErrorLog("dbrefresh PRAGMA journal_mode create fail, ", err)
		return err
	}
	_, err = RefreshDB.Exec("CREATE INDEX inx_TaskId ON refreshinfo(TaskId);")
	if err != nil {
		fmt.Println("create refreshdb index taskid error, ", err)
		return err
	}
	_, err = RefreshDB.Exec("CREATE INDEX inx_Status ON refreshinfo(Status);")
	if err != nil {
		fmt.Println("create refreshdb index status error, ", err)
		return err
	}
	err = Updaterefreshdb(0, 0, "")
	if err != nil {
		logger.ErrorLog("dbrefresh update status fail, ", err)
		return err
	}
	//创建预热表
	PreheatDB, err = sql.Open("sqlite3", "/home/icache/hcs/hcs/db/preheat.db")
	//defer PreheatDB1.Close()
	if err != nil {
		logger.ErrorLog("dbpreheat init fail, ", err)
		return err
	}
	rows1, err := PreheatDB.Query("select name from sqlite_master where type='table' order by name")
	if err != nil {
		logger.ErrorLog("query preheat init fail, ", err)
		return err
	}
	if rows1.Next() {
		rows1.Close()
		_, err = PreheatDB.Exec("DROP INDEX inx_TaskId;")
		if err != nil {
			logger.ErrorLog("delete preheatdb index TaskId error, ", err)
			return err
		}
		_, err = PreheatDB.Exec("DROP INDEX inx_Status;")
		if err != nil {
			logger.ErrorLog("delete preheatdb index status error, ", err)
			return err
		}
	} else {
		rows1.Close()
	}
	preheatinfo := `
    CREATE TABLE IF NOT EXISTS preheatinfo(
        Uid INTEGER PRIMARY KEY ,
        TaskId VARCHAR(64) NOT NULL,
        ResUrl VARCHAR(200) NOT NULL,
 		AccessUrl VARCHAR(200) NULL,
 		Md5 VARCHAR(20) NULL,
        UrlType VARCHAR(2) NOT NULL,
		ResKey VARCHAR(50) NULL,
		ScopeType  VARCHAR(2) NULL,
		FleeSwitch VARCHAR(2) NULL,
		FleeIp VARCHAR(20) NULL,
		BackSourceIp VARCHAR(20) NULL,
		Hot VARCHAR(2) NULL,
		RangeBegin VARCHAR(5) NULL,
		RangeEnd VARCHAR(5) NULL,
		Levels NUMBER NOT NULL, 
		Status NUMBER	NOT NULL,
		Retrytimes NUMBER NULL, 
		Times NUMBER  NOT NULL
    );`
	_, err = PreheatDB.Exec(preheatinfo)
	if err != nil {
		logger.ErrorLog("dbpreheat create fail, ", err)
		return err
	}
	_, err = PreheatDB.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		logger.ErrorLog("dbpreheat PRAGMA journal_mode create fail, ", err)
		return err
	}
	_, err = PreheatDB.Exec("CREATE INDEX inx_TaskId ON preheatinfo(TaskId);")
	if err != nil {
		logger.ErrorLog("create preheatdb index taskid error, ", err)
		return err
	}
	_, err = PreheatDB.Exec("CREATE INDEX inx_Status ON preheatinfo(Status);")
	if err != nil {
		logger.ErrorLog("create preheatdb index status error, ", err)
		return err
	}
	err = Updatepreheatdb(0, 0, "")
	if err != nil {
		logger.ErrorLog("dbpreheat update status fail, ", err)
		return err
	}
	////创建上报状态表
	//ReprotDB,err=sql.Open("sqlite3", "/home/icache/hcs/hcs/db/report.db")
	////defer ReprotDB1.Close()
	//if err!=nil{
	//	logger.ErrorLog("dbreport init fail, ",err)
	//	return err
	//}
	//rows2,err := ReprotDB.Query("select name from sqlite_master where type='table' order by name")
	//if err!=nil{
	//	logger.ErrorLog("query report init fail, ",err)
	//	return err
	//}
	//if rows2.Next(){
	//	rows2.Close()
	//	_, err = ReprotDB.Exec("DROP INDEX inx_taskid;")
	//	if err != nil {
	//		logger.ErrorLog("delete reportdb index error, ",err)
	//		return err
	//	}
	//}else {
	//	rows2.Close()
	//}
	//reportinfo := `
	//CREATE TABLE IF NOT EXISTS reportinfo(
	//    uid INTEGER PRIMARY KEY,
	//    taskid NUMBER NOT NULL,
	//	url VARCHAR(64) NOT NULL,
	//    status NUMBER NOT NULL,
	//	progress NUMBER  NULL,
	//	start_time NUMBER  NOT NULL,
	//	end_time NUMBER  NOT NULL,
	//	err_code NUMBER NULL,
	//	errmsg VARCHAR(200) NULL,
	//	levels NUMBER NOT NULL
	//);`
	//_,err=ReprotDB.Exec(reportinfo)
	//if err!=nil{
	//	logger.ErrorLog("dbreport create fail, ",err)
	//	return err
	//}
	//_,err=ReprotDB.Exec("PRAGMA journal_mode=WAL;")
	//if err!=nil{
	//	logger.ErrorLog("dbreport PRAGMA journal_mode create fail, ",err)
	//	return err
	//}
	//_, err = ReprotDB.Exec("CREATE INDEX inx_taskid ON reportinfo(taskid);")
	//if err != nil {
	//	logger.ErrorLog("create reportdb index error, ",err)
	//	return err
	//}

	logger.InfoLog("db init success")
	return nil
}

//插入刷新
func Insertrefreshdb(task commonstruct.DbInsertStruct) error {
	if task.Num != len(task.Tasks) {
		return errors.New("taskNum is error")
	}
	refreshlock.Lock()
	defer refreshlock.Unlock()
	num := task.Num / 100
	rem := task.Num % 100
	times := time.Now().Unix()
	if num != 0 {
		for i := 0; i < num; i++ {
			sqlStr := "insert into refreshinfo(TaskId,ResUrl,AccessUrl,Md5,UrlType,ResKey,ScopeType,FleeSwitch,FleeIp,BackSourceIp,Hot,RangeBegin,RangeEnd,Levels,Status,Retrytimes,Times)select"
			for j := 0; j < 100; j++ {
				if j == 99 {
					sqlStr += fmt.Sprintf(" \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%d\", \"%d\", \"%d\", \"%d\";", task.Tasks[i*100+j].TaskId,
						task.Tasks[i*100+j].ResUrl, task.Tasks[i*100+j].AccessUrl, task.Tasks[i*100+j].Md5, task.Tasks[i*100+j].UrlType, task.Tasks[i*100+j].ResKey, task.Tasks[i*100+j].ScopeType,
						task.Tasks[i*100+j].FleeSwitch, task.Tasks[i*100+j].FleeIp, task.Tasks[i*100+j].BackSourceIp, task.Tasks[i*100+j].Hot, task.Tasks[i*100+j].RangeBegin, task.Tasks[i*100+j].RangeEnd,
						task.Tasks[i*100+j].Level, 0, 0, times)
				} else {
					sqlStr += fmt.Sprintf(" \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%d\", \"%d\", \"%d\", \"%d\" union all select", task.Tasks[i*100+j].TaskId,
						task.Tasks[i*100+j].ResUrl, task.Tasks[i*100+j].AccessUrl, task.Tasks[i*100+j].Md5, task.Tasks[i*100+j].UrlType, task.Tasks[i*100+j].ResKey, task.Tasks[i*100+j].ScopeType,
						task.Tasks[i*100+j].FleeSwitch, task.Tasks[i*100+j].FleeIp, task.Tasks[i*100+j].BackSourceIp, task.Tasks[i*100+j].Hot, task.Tasks[i*100+j].RangeBegin, task.Tasks[i*100+j].RangeEnd,
						task.Tasks[i*100+j].Level, 0, 0, times)
				}
			}
			_, err := RefreshDB.Exec(sqlStr)
			if err != nil {
				logger.ErrorLog("Insertrefreshdb fail, ", err, ", ", task)
				return err
			}
		}
	}
	if rem != 0 {
		sqlStr := "insert into refreshinfo(TaskId,ResUrl,AccessUrl,Md5,UrlType,ResKey,ScopeType,FleeSwitch,FleeIp,BackSourceIp,Hot,RangeBegin,RangeEnd,Levels,Status,Retrytimes,Times)select"
		for i := 0; i < rem; i++ {
			if i == rem-1 {
				sqlStr += fmt.Sprintf(" \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%d\", \"%d\", \"%d\", \"%d\";", task.Tasks[num*100+i].TaskId,
					task.Tasks[num*100+i].ResUrl, task.Tasks[num*100+i].AccessUrl, task.Tasks[num*100+i].Md5, task.Tasks[num*100+i].UrlType, task.Tasks[num*100+i].ResKey, task.Tasks[num*100+i].ScopeType,
					task.Tasks[num*100+i].FleeSwitch, task.Tasks[num*100+i].FleeIp, task.Tasks[num*100+i].BackSourceIp, task.Tasks[num*100+i].Hot, task.Tasks[num*100+i].RangeBegin, task.Tasks[num*100+i].RangeEnd,
					task.Tasks[num*100+i].Level, 0, 0, times)
			} else {
				sqlStr += fmt.Sprintf(" \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%d\", \"%d\", \"%d\", \"%d\" union all select", task.Tasks[num*100+i].TaskId,
					task.Tasks[num*100+i].ResUrl, task.Tasks[num*100+i].AccessUrl, task.Tasks[num*100+i].Md5, task.Tasks[num*100+i].UrlType, task.Tasks[num*100+i].ResKey, task.Tasks[num*100+i].ScopeType,
					task.Tasks[num*100+i].FleeSwitch, task.Tasks[num*100+i].FleeIp, task.Tasks[num*100+i].BackSourceIp, task.Tasks[num*100+i].Hot, task.Tasks[num*100+i].RangeBegin, task.Tasks[num*100+i].RangeEnd,
					task.Tasks[num*100+i].Level, 0, 0, times)
			}
		}
		_, err := RefreshDB.Exec(sqlStr)
		if err != nil {
			logger.ErrorLog("Insertrefreshdb fail, ", err, ", ", task)
			return err
		}
	}
	//logger.InfoLog("Insertpreheatdb sucess. task: ", task)
	return nil
}

//插入预热
func Insertpreheatdb(task commonstruct.DbInsertStruct) error {
	if task.Num != len(task.Tasks) {
		return errors.New("taskNum is error")
	}
	preheatlock.Lock()
	defer preheatlock.Unlock()
	num := task.Num / 100
	rem := task.Num % 100
	times := time.Now().Unix()
	if num != 0 {
		for i := 0; i < num; i++ {
			sqlStr := "insert into preheatinfo(TaskId,ResUrl,AccessUrl,Md5,UrlType,ResKey,ScopeType,FleeSwitch,FleeIp,BackSourceIp,Hot,RangeBegin,RangeEnd,Levels,Status,Retrytimes,Times)select"
			for j := 0; j < 100; j++ {
				if j == 99 {
					sqlStr += fmt.Sprintf(" \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%d\", \"%d\", \"%d\", \"%d\";", task.Tasks[i*100+j].TaskId,
						task.Tasks[i*100+j].ResUrl, task.Tasks[i*100+j].AccessUrl, task.Tasks[i*100+j].Md5, task.Tasks[i*100+j].UrlType, task.Tasks[i*100+j].ResKey, task.Tasks[i*100+j].ScopeType,
						task.Tasks[i*100+j].FleeSwitch, task.Tasks[i*100+j].FleeIp, task.Tasks[i*100+j].BackSourceIp, task.Tasks[i*100+j].Hot, task.Tasks[i*100+j].RangeBegin, task.Tasks[i*100+j].RangeEnd,
						task.Tasks[i*100+j].Level, 0, 0, times)
				} else {
					sqlStr += fmt.Sprintf(" \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%d\", \"%d\", \"%d\", \"%d\" union all select", task.Tasks[i*100+j].TaskId,
						task.Tasks[i*100+j].ResUrl, task.Tasks[i*100+j].AccessUrl, task.Tasks[i*100+j].Md5, task.Tasks[i*100+j].UrlType, task.Tasks[i*100+j].ResKey, task.Tasks[i*100+j].ScopeType,
						task.Tasks[i*100+j].FleeSwitch, task.Tasks[i*100+j].FleeIp, task.Tasks[i*100+j].BackSourceIp, task.Tasks[i*100+j].Hot, task.Tasks[i*100+j].RangeBegin, task.Tasks[i*100+j].RangeEnd,
						task.Tasks[i*100+j].Level, 0, 0, times)
				}
			}
			_, err := PreheatDB.Exec(sqlStr)
			if err != nil {
				logger.ErrorLog("Insertpreheatdb fail, ", err, ", ", task)
				return err
			}
		}
	}
	if rem != 0 {
		sqlStr := "insert into preheatinfo(TaskId,ResUrl,AccessUrl,Md5,UrlType,ResKey,ScopeType,FleeSwitch,FleeIp,BackSourceIp,Hot,RangeBegin,RangeEnd,Levels,Status,Retrytimes,Times)select"
		for i := 0; i < rem; i++ {
			if i == rem-1 {
				sqlStr += fmt.Sprintf(" \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%d\", \"%d\", \"%d\", \"%d\";", task.Tasks[num*100+i].TaskId,
					task.Tasks[num*100+i].ResUrl, task.Tasks[num*100+i].AccessUrl, task.Tasks[num*100+i].Md5, task.Tasks[num*100+i].UrlType, task.Tasks[num*100+i].ResKey, task.Tasks[num*100+i].ScopeType,
					task.Tasks[num*100+i].FleeSwitch, task.Tasks[num*100+i].FleeIp, task.Tasks[num*100+i].BackSourceIp, task.Tasks[num*100+i].Hot, task.Tasks[num*100+i].RangeBegin, task.Tasks[num*100+i].RangeEnd,
					task.Tasks[num*100+i].Level, 0, 0, times)
			} else {
				sqlStr += fmt.Sprintf(" \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%d\", \"%d\", \"%d\", \"%d\" union all select", task.Tasks[num*100+i].TaskId,
					task.Tasks[num*100+i].ResUrl, task.Tasks[num*100+i].AccessUrl, task.Tasks[num*100+i].Md5, task.Tasks[num*100+i].UrlType, task.Tasks[num*100+i].ResKey, task.Tasks[num*100+i].ScopeType,
					task.Tasks[num*100+i].FleeSwitch, task.Tasks[num*100+i].FleeIp, task.Tasks[num*100+i].BackSourceIp, task.Tasks[num*100+i].Hot, task.Tasks[num*100+i].RangeBegin, task.Tasks[num*100+i].RangeEnd,
					task.Tasks[num*100+i].Level, 0, 0, times)
			}
		}
		_, err := PreheatDB.Exec(sqlStr)
		if err != nil {
			logger.ErrorLog("Insertpreheatdb fail, ", err, ", ", task)
			return err
		}
	}
	//logger.InfoLog("Insertpreheatdb sucess. task: ", task)
	return nil
}

//插入上报
//func Insertreportdb(task commonstruct.ReportStatus) error{
//	reportlock.Lock()
//	defer reportlock.Unlock()
//	sqlStr := fmt.Sprintf("INSERT INTO reportinfo(taskid,url,status,progress,start_time,end_time,err_code,errmsg,levels) VALUES (\"%d\",\"%s\",\"%d\",\"%d\",\"%d\",\"%d\",\"%d\",\"%s\",\"%d\");",
//		task.SubTaskId,task.Url,task.Status,task.Progress,task.Start_time,task.End_time,task.Err_code,task.Err_Msg,task.Level)
//	_,err:=ReprotDB.Exec(sqlStr)
//	if err != nil {
//		logger.ErrorLog("Insertreportdb fail, ",err)
//		return err
//	}
//	logger.InfoLog("Insertreportdb success, reporttask: ",task.SubTaskId)
//	return nil
//}
//启动刷新恢复
func Updaterefreshdb(retryTimes int, status int, taskid string) error {
	refreshlock.Lock()
	defer refreshlock.Unlock()
	if taskid == "" {
		sqlStr := fmt.Sprintf("update refreshinfo set Status=\"%d\",Retrytimes=\"%d\" where status=1;", status, retryTimes)
		_, err := RefreshDB.Exec(sqlStr)
		if err != nil {
			logger.ErrorLog("update refreshdb status fail, ", err)
			return err
		}
		sqlStr1 := fmt.Sprintf("update refreshinfo set Status=\"%d\",Retrytimes=\"%d\" where status=2;", status, retryTimes)
		_, err = RefreshDB.Exec(sqlStr1)
		if err != nil {
			logger.ErrorLog("update refreshdb status fail, ", err)
			return err
		}
	} else {
		sqlStr := fmt.Sprintf("update refreshinfo set Status=\"%d\",Retrytimes=\"%d\" where TaskId=\"%s\";", status, retryTimes, taskid)
		_, err := RefreshDB.Exec(sqlStr)
		if err != nil {
			logger.ErrorLog("update refreshdb status fail, ", err)
			return err
		}
	}
	return nil
}
func Finishedrefreshdb(task commonstruct.DbDelete) error {
	refreshlock.Lock()
	defer refreshlock.Unlock()
	num := task.Num / 100
	rem := task.Num % 100
	if num != 0 {
		for i := 0; i < num; i++ {
			sqlStr := "update refreshinfo set status=2 where TaskId IN ("
			for j := 0; j < 100; j++ {
				if j == 99 {
					sqlStr += fmt.Sprintf("\"%s\");", task.TaskId[i*100+j])
				} else {
					sqlStr += fmt.Sprintf("\"%s\",", task.TaskId[i*100+j])
				}
			}
			_, err := RefreshDB.Exec(sqlStr)
			if err != nil {
				logger.ErrorLog("delete refreshdb fail, ", err)
				return err
			}
		}
	}
	if rem != 0 {
		sqlStr := "update refreshinfo set status=2 where TaskId IN ("
		for i := 0; i < rem; i++ {
			if i == rem-1 {
				sqlStr += fmt.Sprintf("\"%s\");", task.TaskId[num*100+i])
			} else {
				sqlStr += fmt.Sprintf("\"%s\",", task.TaskId[num*100+i])
			}
		}
		_, err := RefreshDB.Exec(sqlStr)
		if err != nil {
			logger.ErrorLog("delete refreshdb fail, ", err)
			return err
		}
	}
	return nil
}

//启动预热恢复
func Updatepreheatdb(retryTimes int, status int, taskid string) error {
	preheatlock.Lock()
	defer preheatlock.Unlock()
	if taskid == "" {
		sqlStr := fmt.Sprintf("update preheatinfo set Status=\"%d\",Retrytimes=\"%d\" where status=1;", status, retryTimes)
		_, err := PreheatDB.Exec(sqlStr)
		if err != nil {
			logger.ErrorLog("update preheat status fail, ", err)
			return err
		}
		sqlStr1 := fmt.Sprintf("update preheatinfo set Status=\"%d\",Retrytimes=\"%d\" where status=2;", status, retryTimes)
		_, err = PreheatDB.Exec(sqlStr1)
		if err != nil {
			logger.ErrorLog("update preheat status fail, ", err)
			return err
		}
	} else {
		sqlStr := fmt.Sprintf("update preheatinfo set Status=\"%d\",Retrytimes=\"%d\" where TaskId=\"%s\";", status, retryTimes, taskid)
		_, err := PreheatDB.Exec(sqlStr)
		if err != nil {
			logger.ErrorLog("update preheat status fail, ", err)
			return err
		}
	}
	return nil
}
func Finishedpreheatdb(task commonstruct.DbDelete) error {
	preheatlock.Lock()
	defer preheatlock.Unlock()
	num := task.Num / 100
	rem := task.Num % 100
	if num != 0 {
		for i := 0; i < num; i++ {
			sqlStr := "update preheatinfo set status=2 where TaskId IN ("
			for j := 0; j < 100; j++ {
				if j == 99 {
					sqlStr += fmt.Sprintf("\"%s\");", task.TaskId[i*100+j])
				} else {
					sqlStr += fmt.Sprintf("\"%s\",", task.TaskId[i*100+j])
				}
			}
			_, err := PreheatDB.Exec(sqlStr)
			if err != nil {
				logger.ErrorLog("delete preheatdb fail, ", err)
				return err
			}
		}
	}
	if rem != 0 {
		sqlStr := "update preheatinfo set status=2 where TaskId IN ("
		for i := 0; i < rem; i++ {
			if i == rem-1 {
				sqlStr += fmt.Sprintf("\"%s\");", task.TaskId[num*100+i])
			} else {
				sqlStr += fmt.Sprintf("\"%s\",", task.TaskId[num*100+i])
			}
		}
		_, err := PreheatDB.Exec(sqlStr)
		if err != nil {
			logger.ErrorLog("delete preheatdb fail, ", err)
			return err
		}
	}
	return nil
}

//刷新查询
func Queryrefreshdb() (commonstruct.DbTempReqRefreshStruct, error) {
	var refreshtask commonstruct.DbTempReqRefreshStruct
	refreshlock.Lock()
	defer refreshlock.Unlock()
	rows, err := RefreshDB.Query("select TaskId,ResUrl,AccessUrl,UrlType,Levels,Retrytimes from refreshinfo where status=0 limit 100")
	if err != nil {
		logger.ErrorLog("Queryrefreshdb fail, ", err)
		return refreshtask, err
	}
	defer rows.Close()
	for rows.Next() {
		var subrefreshtask commonstruct.TempReqRefreshStruct
		subrefreshtask.ReqRefreshTask = make([]commonstruct.ReqRefreshStruct, 1)
		err := rows.Scan(&subrefreshtask.ReqRefreshTask[0].TaskId, &subrefreshtask.ReqRefreshTask[0].ResUrl, &subrefreshtask.AccessUrl, &subrefreshtask.ReqRefreshTask[0].UrlType, &subrefreshtask.Level, &subrefreshtask.RetryTimes)
		if err != nil {
			logger.ErrorLog("Queryrefreshdb fail, ", err)
			return refreshtask, err
		}
		refreshtask.Num++
		refreshtask.RefreshTask = append(refreshtask.RefreshTask, subrefreshtask)
	}
	num := refreshtask.Num / 100
	rem := refreshtask.Num % 100
	if num != 0 {
		for i := 0; i < num; i++ {
			sqlStr := "update refreshinfo set status=1 where TaskId IN ("
			for j := 0; j < 100; j++ {
				if j == 99 {
					sqlStr += fmt.Sprintf("\"%s\");", refreshtask.RefreshTask[i*100+j].ReqRefreshTask[0].TaskId)
				} else {
					sqlStr += fmt.Sprintf("\"%s\",", refreshtask.RefreshTask[i*100+j].ReqRefreshTask[0].TaskId)
				}
			}
			_, err := RefreshDB.Exec(sqlStr)
			if err != nil {
				logger.ErrorLog("update refreshdb fail, ", err)
				return refreshtask, err
			}
		}
	}
	if rem != 0 {
		sqlStr := "update refreshinfo set status=1 where TaskId IN("
		for i := 0; i < rem; i++ {
			if i == rem-1 {
				sqlStr += fmt.Sprintf("\"%s\");", refreshtask.RefreshTask[num*100+i].ReqRefreshTask[0].TaskId)
			} else {
				sqlStr += fmt.Sprintf("\"%s\",", refreshtask.RefreshTask[num*100+i].ReqRefreshTask[0].TaskId)
			}
		}
		_, err := RefreshDB.Exec(sqlStr)
		if err != nil {
			logger.ErrorLog("update refreshdb fail, ", err)
			return refreshtask, err
		}
	}
	//if refreshtask.Num != 0{
	//	logger.InfoLog("Queryrefreshdb success, puttask: ",refreshtask)
	//}
	return refreshtask, err
}

//查询总数
func QueryCount() int64 {
	preheatlock.Lock()
	defer preheatlock.Unlock()
	rows, err := PreheatDB.Query("select count(*) from preheatinfo")
	if err != nil {
		logger.ErrorLog("query preheatdb task count failed")
		return -1
	}
	defer rows.Close()
	var num1 int64
	if rows.Next() {
		err := rows.Scan(&num1)
		if err != nil {
			logger.ErrorLog("query refreshdb task num1 failed")
			return -1
		}
	}
	logger.InfoLog("preheatdb num is: ", num1)
	refreshlock.Lock()
	defer refreshlock.Unlock()
	rows1, err := RefreshDB.Query("select count(*) from refreshinfo")
	if err != nil {
		logger.ErrorLog("query refreshdb task count failed")
		return -1
	}
	defer rows1.Close()
	var num2 int64
	if rows1.Next() {
		err := rows1.Scan(&num2)
		if err != nil {
			logger.ErrorLog("query refreshdb task num2 failed")
			return -1
		}
	}
	logger.InfoLog("refreshdb num is: ", num2)
	if num2 > num1 {
		return num2
	} else {
		return num1
	}
	return 0
}
func QueryPreheatCount() int {
	preheatlock.Lock()
	defer preheatlock.Unlock()
	rows, err := PreheatDB.Query("select count(*) from preheatinfo")
	if err != nil {
		logger.ErrorLog("query preheatdb task count failed")
		return 0
	}
	defer rows.Close()
	var num int
	if rows.Next() {
		err := rows.Scan(&num)
		if err != nil {
			logger.ErrorLog(err)
			return num
		}
	}
	return num
}

//预热查询
func Querypreheatdb() (commonstruct.DbTempReqPreheatStruct, error) {
	var preheattask commonstruct.DbTempReqPreheatStruct
	preheatlock.Lock()
	defer preheatlock.Unlock()
	rows, err := PreheatDB.Query("select TaskId,ResUrl,AccessUrl,Md5,UrlType,ResKey,ScopeType,FleeSwitch,FleeIp,BackSourceIp,Hot,RangeBegin,Levels,Retrytimes from preheatinfo where status=0 limit 100")
	if err != nil {
		logger.ErrorLog("Querypreheatdb fail, ", err)
		return preheattask, err
	}
	defer rows.Close()
	for rows.Next() {
		var subpreheattask commonstruct.TempReqPreheatStruct
		subpreheattask.ReqPreheatTask = make([]commonstruct.ReqPreheatStruct, 1)
		err := rows.Scan(&subpreheattask.ReqPreheatTask[0].TaskId, &subpreheattask.ReqPreheatTask[0].ResUrl, &subpreheattask.ReqPreheatTask[0].AccessUrl,
			&subpreheattask.ReqPreheatTask[0].Md5, &subpreheattask.ReqPreheatTask[0].UrlType, &subpreheattask.ReqPreheatTask[0].ResKey, &subpreheattask.ReqPreheatTask[0].ScopeType,
			&subpreheattask.ReqPreheatTask[0].FleeSwitch, &subpreheattask.ReqPreheatTask[0].FleeIp, &subpreheattask.ReqPreheatTask[0].BackSourceIp, &subpreheattask.ReqPreheatTask[0].Hot,
			&subpreheattask.ReqPreheatTask[0].RangeBegin, &subpreheattask.Level, &subpreheattask.RetryTimes)
		if err != nil {
			logger.ErrorLog(err)
			return preheattask, err
		}
		preheattask.Num++
		preheattask.PreheatTask = append(preheattask.PreheatTask, subpreheattask)
	}
	num := preheattask.Num / 100
	rem := preheattask.Num % 100
	if num != 0 {
		for i := 0; i < num; i++ {
			sqlStr := "update preheatinfo set status=1 where TaskId IN ("
			for j := 0; j < 100; j++ {
				if j == 99 {
					sqlStr += fmt.Sprintf("\"%s\");", preheattask.PreheatTask[i*100+j].ReqPreheatTask[0].TaskId)
				} else {
					sqlStr += fmt.Sprintf("\"%s\",", preheattask.PreheatTask[i*100+j].ReqPreheatTask[0].TaskId)
				}
			}
			_, err := PreheatDB.Exec(sqlStr)
			if err != nil {
				logger.ErrorLog("update preheatdb fail, ", err)
				return preheattask, err
			}
		}
	}
	if rem != 0 {
		sqlStr := "update preheatinfo set status=1 where TaskId IN("
		for i := 0; i < rem; i++ {
			if i == rem-1 {
				sqlStr += fmt.Sprintf("\"%s\");", preheattask.PreheatTask[num*100+i].ReqPreheatTask[0].TaskId)
			} else {
				sqlStr += fmt.Sprintf("\"%s\",", preheattask.PreheatTask[num*100+i].ReqPreheatTask[0].TaskId)
			}
		}
		_, err := PreheatDB.Exec(sqlStr)
		if err != nil {
			logger.ErrorLog("update preheatdb fail, ", err)
			return preheattask, err
		}
	}
	//if preheattask.Num != 0{
	//	logger.InfoLog("Querypreheatdb success, puttask: ",preheattask)
	//}
	return preheattask, err
}

//上报查询
func Queryreportdb() (commonstruct.ReportList, error) {
	reportlock.Lock()
	defer reportlock.Unlock()
	var subreporttask commonstruct.ReportList
	if ReportTask.Size() == 0 {
		return subreporttask, nil
	}
	for _, entry := range ReportTask.ToSlice() {
		k, v := entry.Key(), entry.Value()
		if k == nil {
			logger.ErrorLog("iterater reporttask error", k, v)
			return subreporttask, errors.New("error")
		}
		task := v.(commonstruct.ReportStatus)
		if subreporttask.TaskNum > maxReportTaskNum-1 {
			break
		}
		subreporttask.TaskNum++
		subreporttask.Tasks = append(subreporttask.Tasks, task)
	}
	return subreporttask, nil
}

//上报查询
func QueryOvertimereportdb() (commonstruct.ReportList, error) {
	reportlock.Lock()
	defer reportlock.Unlock()
	var subreporttask commonstruct.ReportList
	if ReportTask.Size() == 0 {
		return subreporttask, nil
	}
	for _, entry := range ReportTask.ToSlice() {
		k, v := entry.Key(), entry.Value()
		if k == nil {
			logger.ErrorLog("iterater reporttask error", k, v)
			return subreporttask, errors.New("error")
		}
		task := v.(commonstruct.ReportStatus)
		if time.Now().Unix()-task.End_time < 60 {
			continue
		}
		subreporttask.TaskNum++
		subreporttask.Tasks = append(subreporttask.Tasks, task)
	}
	//上报最大数量
	logger.InfoLog("Queryreportdb success, reporttask: ", subreporttask)
	return subreporttask, nil
}

//上报查询是否存在
func UpdateOrInsertreportdb(task commonstruct.ReportStatus) error {
	//reportlock.Lock()
	val, err := ReportTask.Get(task.SubTaskId)
	if err != nil {
		str := fmt.Sprintf("find taskid: %d from reporttask failed", task.SubTaskId)
		logger.ErrorLog(str)
		return err
	}
	if val != nil {
		if val.(commonstruct.ReportStatus).Status == 3 {
			//任务已经成功无需在更新
			return nil
		}
		//找到了就更新
		_, err = ReportTask.Replace(task.SubTaskId, task)
		if err != nil {
			str := fmt.Sprintf("replace taskid: %d, from reporttask failed", task.SubTaskId)
			logger.ErrorLog(str)
			return err
		}
		return nil
	}
	_, err = ReportTask.Put(task.SubTaskId, task)
	if err != nil {
		str := fmt.Sprintf("insert taskid: %d into reporttask failed", task.SubTaskId)
		logger.ErrorLog(str)
		return err
	}
	return nil
}

//更新上报
//func Updatereportdb(subreporttask commonstruct.ReportStatus)(error) {
//	reportlock.Lock()
//	defer reportlock.Unlock()
//	sqlStr := fmt.Sprintf("update reportinfo set status=\"%d\",progress=\"%d\",start_time=\"%d\",end_time=\"%d\",err_code=\"%d\",errmsg=\"%s\" where taskid=\"%d\";",
//		subreporttask.Status, subreporttask.Progress,subreporttask.Start_time,subreporttask.End_time,subreporttask.Err_code,subreporttask.Err_Msg,subreporttask.SubTaskId)
//	_,err :=ReprotDB.Exec(sqlStr)
//	if err!=nil{
//		logger.ErrorLog("update report status fail, ",err)
//		return err
//	}
//	logger.InfoLog("updatereport success, reporttask: ",subreporttask)
//	return  err
//}
//查询超时刷新任务并删除
func QueryOvertimeRefresh() commonstruct.DbDelete {
	refreshlock.Lock()
	defer refreshlock.Unlock()
	var task commonstruct.DbDelete
	rows, err := RefreshDB.Query("select TaskId, Times from refreshinfo")
	if err != nil {
		logger.ErrorLog("Queryrefresh fail, ", err)
		return task
	}
	defer rows.Close()
	var times int64
	var taskid string
	for rows.Next() {
		err := rows.Scan(&taskid, &times)
		if err != nil {
			logger.ErrorLog("Queryrefresh fail, ", err)
			return task
		}
		//上报最大数量
		if time.Now().Unix()-times < 600 {
			continue
		}
		task.Num++
		task.TaskId = append(task.TaskId, taskid)
	}
	return task
}

//删除刷新
func Deleterefreshdb(task commonstruct.DbDelete) error {
	refreshlock.Lock()
	defer refreshlock.Unlock()
	num := task.Num / 100
	rem := task.Num % 100
	if num != 0 {
		for i := 0; i < num; i++ {
			sqlStr := "delete from refreshinfo where TaskId IN ("
			for j := 0; j < 100; j++ {
				if j == 99 {
					sqlStr += fmt.Sprintf("\"%s\");", task.TaskId[i*100+j])
				} else {
					sqlStr += fmt.Sprintf("\"%s\",", task.TaskId[i*100+j])
				}
			}
			_, err := RefreshDB.Exec(sqlStr)
			if err != nil {
				logger.ErrorLog("delete refreshdb fail, ", err)
				return err
			}
		}
	}
	if rem != 0 {
		sqlStr := "delete from refreshinfo where TaskId IN ("
		for i := 0; i < rem; i++ {
			if i == rem-1 {
				sqlStr += fmt.Sprintf("\"%s\");", task.TaskId[num*100+i])
			} else {
				sqlStr += fmt.Sprintf("\"%s\",", task.TaskId[num*100+i])
			}
		}
		_, err := RefreshDB.Exec(sqlStr)
		if err != nil {
			logger.ErrorLog("delete refreshdb fail, ", err)
			return err
		}
	}
	return nil
}
func QueryOvertimePreheat() commonstruct.DbDelete {
	preheatlock.Lock()
	defer preheatlock.Unlock()
	var task commonstruct.DbDelete
	rows, err := PreheatDB.Query("select TaskId, Times from preheatinfo")
	if err != nil {
		logger.ErrorLog("Querypreheat fail, ", err)
		return task
	}
	defer rows.Close()
	var times int64
	var taskid string
	for rows.Next() {
		err := rows.Scan(&taskid, &times)
		if err != nil {
			logger.ErrorLog("Querypreheat fail, ", err)
			return task
		}
		//上报最大数量
		if time.Now().Unix()-times < 600 {
			continue
		}
		task.Num++
		task.TaskId = append(task.TaskId, taskid)
	}
	return task
}

//删除预热
func Deletepreheatdb(task commonstruct.DbDelete) error {
	preheatlock.Lock()
	defer preheatlock.Unlock()
	num := task.Num / 100
	rem := task.Num % 100
	if num != 0 {
		for i := 0; i < num; i++ {
			sqlStr := "delete from preheatinfo where TaskId IN ("
			for j := 0; j < 100; j++ {
				if j == 99 {
					sqlStr += fmt.Sprintf("\"%s\");", task.TaskId[i*100+j])
				} else {
					sqlStr += fmt.Sprintf("\"%s\",", task.TaskId[i*100+j])
				}
			}
			_, err := PreheatDB.Exec(sqlStr)
			if err != nil {
				logger.ErrorLog("delete preheatdb fail, ", err)
				return err
			}
		}
	}
	if rem != 0 {
		sqlStr := "delete from preheatinfo where TaskId IN ("
		for i := 0; i < rem; i++ {
			if i == rem-1 {
				sqlStr += fmt.Sprintf("\"%s\");", task.TaskId[num*100+i])
			} else {
				sqlStr += fmt.Sprintf("\"%s\",", task.TaskId[num*100+i])
			}
		}
		_, err := PreheatDB.Exec(sqlStr)
		if err != nil {
			logger.ErrorLog("delete preheatdb fail, ", err)
			return err
		}
	}
	return nil
}

//删除上报
func Deletereportdb(task commonstruct.DbDelete) error {
	reportlock.Lock()
	defer reportlock.Unlock()
	for _, val := range task.TaskId {
		taskid, err := strconv.Atoi(val)
		if err != nil {
			logger.ErrorLog("atoi err, taskid: ", val)
			return err
		}
		_, err = ReportTask.Remove(taskid)
		if err != nil {
			logger.ErrorLog("delete reporttask err, taskid: ", val)
			return err
		}
	}
	return nil
}

//删除
func Deletetaskdb(reporttask commonstruct.ReportList) error {
	var deleteTask commonstruct.DbDelete
	for i := range reporttask.Tasks {
		if reporttask.Tasks[i].Status == 3 || reporttask.Tasks[i].Status == 4 || reporttask.Tasks[i].Status != 2 {
			deleteTask.Num++
			deleteTask.TaskId = append(deleteTask.TaskId, strconv.Itoa(reporttask.Tasks[i].SubTaskId))
		}
	}
	//go Finishedpreheatdb(deleteTask)

	//go Finishedrefreshdb(deleteTask)

	err := Deleterefreshdb(deleteTask)
	if err != nil {
		return err
	}
	err = Deletepreheatdb(deleteTask)
	if err != nil {
		return err
	}
	err = Deletereportdb(deleteTask)
	if err != nil {
		return err
	}
	//logger.InfoLog("delete task success: ", deleteTask)
	return nil
}

//分组信息插入
func Insertdevicedb(deviceinfo commonstruct.Groupinfo) {
	Devicelock.Lock()
	defer Devicelock.Unlock()
	tmpDeviceInfo := DeviceInfo
	DeviceInfo.DevInfoList = nil
	for i := range deviceinfo.DevInfoList {
		DeviceInfo.DevInfoList = append(DeviceInfo.DevInfoList, commonstruct.DevInfo{deviceinfo.DevInfoList[i].DevId, deviceinfo.DevInfoList[i].DevIp, deviceinfo.DevInfoList[i].Stauts})
	}
	DeviceInfo.PopId = deviceinfo.PopId
	DeviceInfo.GroupiId = deviceinfo.GroupiId
	logger.InfoLog("Insertdevicedb success, ", DeviceInfo)
	if 0 == len(tmpDeviceInfo.DevInfoList) {
		return
	}
	if tmpDeviceInfo.PopId != deviceinfo.PopId || tmpDeviceInfo.GroupiId != deviceinfo.GroupiId {
		masterPath := "/master_" + strconv.Itoa(deviceinfo.GroupiId) + strconv.Itoa(deviceinfo.PopId)
		err, zkIp := parsefile.ParseXml()
		if err != nil {
			logger.ErrorLog("Get zkIp error, ", err)
			return
		}
		zkConfig := &zkelectmaster.ZookeeperConfig{
			Servers:    zkIp,
			RootPath:   "/cmsAgent",
			MasterPath: masterPath,
		}
		zkelectmaster.OnNetworkChange(zkConfig)
	}
}

//更新分组信息
func Updatedevicedb(status int, devip string) {
	Devicelock.Lock()
	defer Devicelock.Unlock()
	for i := range DeviceInfo.DevInfoList {
		if devip == DeviceInfo.DevInfoList[i].DevIp {
			DeviceInfo.DevInfoList[i].Stauts = status
			//logger.InfoLog("update deviceinfo success, devip: ",devip, ", status: ", status)
		}
	}
}

//分组信息查询
func Querydevicedb() commonstruct.Groupinfo {
	var deviceinfo commonstruct.Groupinfo
	Devicelock.Lock()
	defer Devicelock.Unlock()
	deviceinfo = DeviceInfo
	return deviceinfo
}

//获取最优设备
func GetBestDevice() (commonstruct.Groupinfo, error) {
	Devicelock.Lock()
	defer Devicelock.Unlock()
	var deviceinfo commonstruct.Groupinfo
	if len(DeviceInfo.DevInfoList) == 0 {
		return deviceinfo, errors.New("no data")
	}
	optimalDev := DeviceInfo.DevInfoList[0]
	for i := range DeviceInfo.DevInfoList {
		if DeviceInfo.DevInfoList[i].Stauts < optimalDev.Stauts {
			optimalDev = DeviceInfo.DevInfoList[i]
		}
	}
	deviceinfo.GroupiId = DeviceInfo.GroupiId
	deviceinfo.PopId = DeviceInfo.PopId
	deviceinfo.DevInfoList = append(deviceinfo.DevInfoList, optimalDev)
	return deviceinfo, nil
}
func QueryDeviceCount() int {
	Devicelock.Lock()
	defer Devicelock.Unlock()
	return len(DeviceInfo.DevInfoList)
}

//分组信息删除
//func Deletedevicedb()error{
//	Devicelock.Lock()
//	defer Devicelock.Unlock()
//	_,err:=DeviceDB.Exec("delete from deviceinfo")
//	if err!=nil{
//		logger.ErrorLog("Deletedevicedb fail",err)
//		return err
//	}
//	for i,_ = range DeviceInfo.DevInfoList{
//		delete(DeviceInfo.DevInfoList,i)
//	}
//	logger.InfoLog("Deletedevicedb success")
//	return nil
//}
//popid查询
//func QueryPopid()(int,error){
//	var popid int
//	devicelock.Lock()
//	rows,err:=DeviceDB.Query("select popid from deviceinfo limit 1")
//	if err!=nil{
//		logger.ErrorLog("Querydevicedb fail, ",err)
//		devicelock.Unlock()
//		return -1,err
//	}
//	defer rows.Close()
//	//var status int
//	if rows.Next(){
//		rows.Scan(&popid)
//	}else {
//		logger.InfoLog("get popid fail")
//		popid=-1
//	}
//	logger.InfoLog("get popid success, popid:",popid)
//	return popid,nil
//}
//插入待处理队列（为了url去重）
func InsertWaitTaskQueue(subpreheattask commonstruct.TempReqPreheatStruct) error {
	val, _ := WaitPreheatTask.Get(subpreheattask.ReqPreheatTask[0].ResUrl)
	if val != nil {
		var flag int
		switch val.(type) {
		case commonstruct.TempReqPreheatStruct:
			//存在且MD5相同
			if val.(commonstruct.TempReqPreheatStruct).ReqPreheatTask[0].Md5 == subpreheattask.ReqPreheatTask[0].Md5 {
				var tmp []commonstruct.TempReqPreheatStruct
				tmp = append(tmp, val.(commonstruct.TempReqPreheatStruct))
				tmp = append(tmp, subpreheattask)
				logger.InfoLog("URL and MD5 is same:", tmp)
				WaitPreheatTask.Replace(subpreheattask.ReqPreheatTask[0].ResUrl, tmp)
			} else { //存在MD5不同需要放回数据库等待任务结束
				logger.InfoLog("URL is same, but MD5 is different:", subpreheattask)
				Updatepreheatdb(subpreheattask.RetryTimes, 0, subpreheattask.ReqPreheatTask[0].TaskId)
			}
		case []commonstruct.TempReqPreheatStruct:
			task := val.([]commonstruct.TempReqPreheatStruct)
			for i := range task {
				if task[i].ReqPreheatTask[0].Md5 == subpreheattask.ReqPreheatTask[0].Md5 {
					task = append(task, subpreheattask)
					logger.InfoLog("URL and MD5 is same:", task)
					WaitPreheatTask.Replace(subpreheattask.ReqPreheatTask[0].ResUrl, task)
					flag = 1
					break
				}
			}
			if flag != 1 {
				//存在MD5不同需要放回数据库等待任务结束
				logger.InfoLog("URL is same, but MD5 is different:", subpreheattask)
				Updatepreheatdb(subpreheattask.RetryTimes, 0, subpreheattask.ReqPreheatTask[0].TaskId)
			}
		default:
			logger.ErrorLog("data type is unknown")
		}
		return errors.New("repeat task")
	} else {
		WaitPreheatTask.Put(subpreheattask.ReqPreheatTask[0].ResUrl, subpreheattask)
	}
	return nil
}

//关闭数据库
func Closedb() {
	RefreshDB.Close()
	PreheatDB.Close()
	//ReprotDB.Close()
	WaitPreheatTask.Clear()
}
