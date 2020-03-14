package conf

import (
	"flag"
	"github.com/larspensjo/config"
	"common/logger"
	"time"
)
var (
	configFile = flag.String("configfile", "/home/icache/hcs/hcs/conf/cms_Agent.conf", "General configuration file")
)

var Config = make(map[string]string)
func init(){
	time.Sleep(1*time.Second)
	flag.Parse()
	//set config file std
	cfg, err := config.ReadDefault(*configFile)
	if err != nil {
			logger.ErrorLog("Fail to find", *configFile, err)
		return
	}
	logger.InfoLog("cms_Agent.conf parse start")
	if cfg.HasSection("config") {
		section, err := cfg.SectionOptions("config")
		if err == nil {
			for _, v := range section {
				options, err := cfg.String("config", v)
				if err == nil {
					Config[v] = options
				}
			}
		}
	}
	logger.InfoLog("cms_Agent.conf init success")
		//vip,err := parsefile.ParsePub()
		//if err != nil{
		//	continue
		//}
		//Config["vip"] = vip
		//logger.RecordLog("read vip success:", Config["vip"])
}