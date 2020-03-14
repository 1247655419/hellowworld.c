// Package oracle ...
// oracle数据库驱动的封装实现
package oracle

import (
    "fmt"
	"sync"
	"errors"

	"common/encrypt"
	"common/util"
	"common/logger"
	"common/config"

	"lib/log4g"
	"database/sql"
	// go-oracle驱动
	_ "github.com/mattn/go-oci8"
)

var once sync.Once

// DB oracle数据库实例
var DB *DataBase

// DBConf 数据库连接信息
type DBConf struct {
    // 用户名
    User string

    // 密码
    PassWord string

    // 数据库地址 ip:port
    Addr string

    // 数据库名
    DBName string
}

// DataBase 数据库实例 单例模式
type DataBase struct {
	// 数据库实例互斥锁
	sync.RWMutex

	// 全局的日志模块
	logger log4g.Logger

    // 全局配置
    conf *config.Config

    // 数据库连接信息
    Oracle DBConf

    // 数据库连接字符串
    // 通过DBConf拼凑
    Addr string
}

// GetInstance 获取数据库实例
func GetInstance() *DataBase {
	once.Do(func() {
		DB = &DataBase{
            logger : logger.GetLogger(),
            conf : config.GetConfigInstance(),
		}
	})

	return DB
}

// Init 数据库初始化 获取配置
func (db *DataBase) Init() {
    rc := db.conf.GetRuntimeConf()

    // 获取oracle配置
    db.Oracle.User = rc.Oracle.User
    db.Oracle.PassWord = rc.Oracle.PassWord
    db.Oracle.Addr = rc.Oracle.Addr
    db.Oracle.DBName = rc.Oracle.DBName

    // 解密oracle密码
    password, err := encrypt.Dncrypt(db.Oracle.PassWord)

    if err != nil {
        db.logger.Fatal("Dao| oracle init failed[%s].", err)
        util.Exit(-1)
    }

	db.Addr = db.Oracle.User + "/" + password + "@" + db.Oracle.Addr + "/" + db.Oracle.DBName 

	return 
}

// GetConnection 创建database连接
func (db *DataBase) GetConnection() (*sql.DB, error) {
	dbSQL, err := sql.Open("oci8", db.Addr)

	if err != nil {
		errmsg := "Dao|oracle Can't connect to database."
		db.logger.Fatal(errmsg)

		return nil, errors.New(errmsg)
	}

    return dbSQL, nil
}

// CloseConnection 关闭数据库连接
func (db *DataBase) CloseConnection(conn *sql.DB) error {
	if conn == nil {
		errmsg := "Dao|oracle Parameter error: conn == nil."
		db.logger.Fatal(errmsg)

		return errors.New(errmsg)
	}

	err := conn.Close()

    if err != nil {
        db.logger.Fatal("Dao|oracle close failed",err)
    }

	return nil
}

// Close 关闭数据库游标及连接
func (db *DataBase) Close(conn *sql.DB, rows *sql.Rows) error {
	if ((conn == nil) || (rows == nil)) {
		errmsg := "Dao|oracle Parameter error: conn == nil or rows == nil."
		db.logger.Fatal(errmsg)

		return errors.New(errmsg)
	}

    err := conn.Close()

    if err != nil {
        db.logger.Fatal(fmt.Sprintf("Dao|oracle conn close failed[%s].", err))

        return err
    }

    err = rows.Close()

    if err != nil {
        db.logger.Fatal(fmt.Sprintf("Dao|oracle rows close failed[%s].", err))

        return err
    }

	return nil
}

// TraverseTable 遍历数据库 获取数据库连接及游标
func (db *DataBase) TraverseTable(sqlStr string) (*sql.DB, *sql.Rows, error) {
	dbSQL, err := sql.Open("oci8", db.Addr)

	if err != nil {
		errmsg := "Dao|oracle Can't connect to database."
		db.logger.Fatal(errmsg)

		return nil, nil, errors.New(errmsg)
	}

	rows, err := dbSQL.Query(sqlStr)

	if err != nil {
        errmsg := fmt.Sprintf("Dao|oracle Failed to query:%s err[%s].", sqlStr, err.Error())
		err1 := dbSQL.Close()

		if err1 != nil {
            db.logger.Fatal(err1)
        }

		db.logger.Fatal(errmsg)

		return nil, nil, errors.New(errmsg)
	}

    return dbSQL, rows, nil
}
