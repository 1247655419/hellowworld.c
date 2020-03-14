package commonstruct

import "strconv"

type BaseConfInfo struct {
	Popid string
	Groupid string

}
//任务拉取结构体
type GetTaskSam struct {
	//任务ID
	SubTaskId int64
	//任务分组ID
	GroupId string
	//任务层级
	Level int
	//任务类型
	TaskType string
	//资源url
	ContentUrl string
	//访问资源url
	AccessUrl string
	//MD5值
	Hash string
	//资源类型
	Type int
	//备用
	InsertTimeStamp int64
	//主备内容中心同步注入
	CdnCenterVip string
}
//任务拉取存储结构体
type RespGetTask struct {
	//cms拆分后的子任务标识
	TaskId string
	//任务类型
	TaskType string
	//预热资源回源URL
	ResUrl string
	//扩展URL，如果存在该URL则用该URL进行ORI计算，使用ResUrl回源
	AccessUrl string
	//预热MD5校验值
	Md5 string
	//URL类型 2：URL；3：DIR；4：FILE；5：索引
	UrlType string
	//预热资源HASH值
	ResKey string
	// 0:在线转离线；1：界面下发的主动注入；2：热点分发的离线任务；3：206 range请求转离线
	ScopeType string
	//逃生通道开关（0关闭，1开启）
	FleeSwitch string
	//逃生通道回源IP
	FleeIp string
	//获取资源的HCS地址，将ResUrl直接代理到该HCS地址进行回源
	BackSourceIp string
	//资源热度
	Hot string
	//206起始位置
	RangeBegin string
	//206结束为止
	RangeEnd string
	Level int//节点级别
}
//批量存入数据库
type DbInsertStruct struct {
	Num int
	Tasks []RespGetTask
}
//响应列表
type RespGetTaskList struct{
	Error_msg string
	TaskNum int // 获取任务数
	Tasks []GetTaskSam
}
//上报状态结构体
type ReportStatus struct {
	SubTaskId int  //CMS拆分后的子任务标识
	Status int	//子任务的执行状态  wating：等待执行（该状态为中间态） complete：执行完毕，且执行结果为成功（该状态为结果态） in-progress：执行中（该状态为中间态） failed：执行失败（该状态为结果态）
	Progress int	//子任务执行进度百分比，如：80代表完成80%
	Start_time int64
	End_time int64
	Err_code int
	Err_Msg string
	Url string
	Level int
}
//状态上报结构体
type ReportList struct {
	VirtualIp string
	TaskNum int
	Tasks []ReportStatus
}
type DevInfo struct {
	DevId int
	DevIp string
	Stauts int//负载能力
}
type Groupinfo struct {
	PopId int
	GroupiId int
	DevInfoList []DevInfo
}
//与HCS之间的预热请求
type ReqPreheatStruct struct {
	//任务id
	TaskId string
	//预热资源回源URL
	ResUrl string
	//扩展URL，如果存在该URL则用该URL进行ORI计算，使用ResUrl回源
	AccessUrl string
	//预热MD5校验值
	Md5 string
	//URL类型 2：URL；3：DIR；4：FILE；5：索引
	UrlType string
	//预热资源HASH值
	ResKey string
	// 0:在线转离线；1：界面下发的主动注入；2：热点分发的离线任务；3：206 range请求转离线
	ScopeType string
	//所属设备di
	SrcDeviceId int
	//所属设备ip
	SrcDeviceIp string
	//逃生通道开关（0关闭，1开启）
	FleeSwitch string
	//逃生通道回源IP
	FleeIp string
	//获取资源的HCS地址，将ResUrl直接代理到该HCS地址进行回源
	BackSourceIp string
	//资源热度
	Hot string
	//206起始位置
	RangeBegin string
	//206结束为止
//	RangeEnd string
	//302强制注入标志
	ForceInject int
}
type TempReqPreheatStruct struct{
	Level int
	RetryTimes int
	OriResUrl string
	ReqPreheatTask []ReqPreheatStruct
}

type DbTempReqPreheatStruct struct{
	Num int
	PreheatTask []TempReqPreheatStruct
}
//HCS返回预热响应
type RespPreheatStruct struct {
	//任务id
	//TaskId string
	//预热资源回源URL
	//ResUrl string
	//结果信息failed， completed， partitial，downloading，sp_error，cache_hit
	ResultMsg string
	//结果-1：failed；0：completed；1：downloading；3：sp_error；4：cache_hit
	Result int
	//预热资源当前缓存大小
	CachedSize int
	//预热资源的ori
	ResOri string
	//预热资源所属设备ip
	DestDeviceIp string
	//预热资源发送IP
	SrcDeviceIp string
	//	//M3U8列表大小
	SubTaskCnt int
	//资源所属id
	DestDeviceId int
	//预热资源HASH值
	ResKey string
	//预热资源发送id
	SrcDeviceId int
	//预热总资源大小
	ResSize string
	//文件列表，以换行结束
	SubTaskUrls []string
}
//刷新请求的结构体
type ReqRefreshStruct struct {
	//任务id
	TaskId string
	//预热资源回源URL
	ResUrl string
	//URL类型 2：URL；3：DIR；4：FILE；5：索引
	UrlType string
	//分组ID
	GroupId string
	//设备id
	DevId string
}
type TempReqRefreshStruct struct{
	Level int
	RetryTimes int
	AccessUrl  string
	ReqRefreshTask []ReqRefreshStruct
}
type DbTempReqRefreshStruct struct{
	Num int
	RefreshTask []TempReqRefreshStruct
}
//刷新响应
type ResqRefreshStruct struct {
	//任务id
	TaskId string
	//结果 -2:"RESOURCE_NOT_FOUND" -1:"DELETE_FAIL" 0:"DELETE_SUCCESS"
	Result int
	//结果信息 "RESOURCE_NOT_FOUND"/"DELETE_FAIL"/"DELETE_SUCCESS"
	ResultMsg string
	//预热资源回源URL
	ResUrl string
	//预热资源的ori
	ResOri string
	//URL类型 2：URL；3：DIR；4：FILE；5：索引
	UrlType string
	//分组ID

}
type DbDelete struct {
	Num int
	TaskId []string
}
type StatusStruct struct{
	Status int `json:"status"`
}
//MD5查询结构体
type Md5QueryStruct struct {
	//任务id号
	TaskId string
	//资源URL
	ResUrl string
	//url类型
	UrlType string
	//reskey
	ResKey string
	//资源热度
	Hot string
	//md5检查开关
	//Checkstr string
	//MD5值
	Md5 string

}
type RespMd5QueryStruct struct {
	//任务id号
	TaskId string
	//MD5检测结果
	Md5Check int
	//reskey
	//ResKey string
	//设备id
	//DeviceId string
	//结果信息
	ResultMsg string
	//结果
	Result int
	//分组ID
	//GroupId string
	//资源类型
	//ResType int
	//资源URL
	ResUrl string
	//ORI值
	ResOri string
	//
	//Priority int
	//
	//BackupDevRatio int

}
// ReportKpi kpi stat data.
type ReportKpi struct {
	TotalTaskNum uint64
	PreheatTaskNum uint64
	PreheatSuccNum uint64
	PreheatFailNum uint64
	RefreshTaskNum uint64
	RefreshSuccNum uint64
	RefreshFailNum uint64
}

func GetInsertSqliteDataStruct(task GetTaskSam)RespGetTask{
	var dbTask RespGetTask
	//数据库需要存储的结构
	if task.Type == 1{
		dbTask.UrlType = "2"
	}
	if task.Type == 2{
		dbTask.UrlType = "3"
	}
	dbTask.ResUrl = task.ContentUrl
	dbTask.TaskId = strconv.FormatInt(task.SubTaskId,10)
	dbTask.Level = task.Level
	dbTask.RangeEnd = "0"
	dbTask.RangeBegin = "0"
	dbTask.Hot = "0"
	dbTask.AccessUrl = task.AccessUrl
	dbTask.FleeIp = ""
	dbTask.FleeSwitch = ""
	dbTask.ScopeType = "1"
	dbTask.ResKey = ""
	dbTask.Md5 = task.Hash
	dbTask.TaskType = task.TaskType
	dbTask.BackSourceIp = task.CdnCenterVip

	return dbTask
}