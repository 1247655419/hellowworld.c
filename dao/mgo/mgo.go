// Package mgo ...
// mongodb驱动mgo的封装实现
package mgo

import (
    "crypto/tls"
    "crypto/x509"
    "errors"
    "fmt"
    "io/ioutil"
    "net"
    "strconv"
    "sync"
    "time"

    "common/config"
    "common/encrypt"
    "common/logger"
    "common/util"
    "structs"

    "lib/log4g"
    "github.com/globalsign/mgo"
    "github.com/globalsign/mgo/bson"
)

var once sync.Once

// DB mongo数据库实例
var DB *DataBase

// 自增序列
var sqTableName = "counter"

// mgo错误标识
const (
    // 重复键值错误
    DuplicateKeyError = "duplicate key error"

    // 未找到数据
    DataNotFound = "not found"
)

// DataBase 数据库实例 单例模式
type DataBase struct {
    // 启停互斥锁
    sync.Mutex

    // 启停标志
    stop bool

    // 启停channel
    stopCh chan struct{}

    // 数据库连接信息 由config模块生成
    dialInfo mgo.DialInfo

    // 探测数据库连接时间间隔
    refreshInterval time.Duration

    // 会话指针
    session *mgo.Session

    // 全局的日志模块
    logger log4g.Logger

    // 全局配置实例
    conf *config.Config
}

// GetInstance 获取数据库实例
func GetInstance() *DataBase {
    once.Do(func() {
        DB = &DataBase{
            logger: logger.GetLogger(),
            conf:   config.GetConfigInstance(),
            stopCh: make(chan struct{}),
        }

        DB.Init()
    })

    return DB
}

// Close 关闭数据库资源
func (db *DataBase) Close() {
    if err := db.session.Ping(); err != nil {
        return
    }

    db.session.Close()
}

// GetDBConf 获取数据库连接配置
func (db *DataBase) GetDBConf() error {
    //获取连接信息
    rc := db.conf.GetRuntimeConf()

    if rc == nil {
        return errors.New("get runtime config failed")
    }

    // begin -----加载SSL证书-----
    clientCertPEM, err := ioutil.ReadFile(util.GetRootDir() + "certificate/mgoClient.pem")

    if err != nil {
        db.logger.Fatal("Dao| ssl mgo mgoClient has err: ", err)
    }

    clientKeyPEM, err := ioutil.ReadFile(util.GetRootDir() + "certificate/mgoClient.key")

    if err != nil {
        db.logger.Fatal("Dao| ssl mgo mgoClient has err: ", err)
    }

    clientCert, err := tls.X509KeyPair(clientCertPEM, clientKeyPEM)

    if err != nil {
        db.logger.Fatal("Dao| ssl mgo clientCert has err: ", err)
    }

    clientCert.Leaf, err = x509.ParseCertificate(clientCert.Certificate[0])

    if err != nil {
        db.logger.Fatal("Dao| ssl mgo clientCert.Leaf has err: ", err)
    }

    caPEM, err := ioutil.ReadFile(util.GetRootDir() + "certificate/mgoCa.pem")

    if err != nil {
        db.logger.Fatal("Dao| ssl mgo caPEM has err: ", err)
    }

    roots := x509.NewCertPool()
    roots.AppendCertsFromPEM(caPEM)
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{clientCert},
        RootCAs:      roots,
        ServerName:   "SAM-MGODB",
        //InsecureSkipVerify: true,
        VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
            for i := 0; i < len(rawCerts); i++ {
                cert, err := x509.ParseCertificate(rawCerts[i])

                if err != nil {
                    db.logger.Error("Dao|VerifyPeerCertificate Error: ", err)

                    continue
                }

                db.logger.Info("Dao|isca,dnsName,subject,notafter,notbefore", cert.IsCA, cert.DNSNames, cert.Subject, cert.NotAfter, cert.NotBefore)
            }

            return nil
        },
    }
    // end -----加载SSL证书-----

    db.dialInfo = mgo.DialInfo{
        Timeout:      rc.MgoConf.Dial.Timeout,
        Database:     rc.MgoConf.Dial.Database,
        Direct:       rc.MgoConf.Dial.Direct,
        Username:     rc.MgoConf.Dial.Username,
        Password:     rc.MgoConf.Dial.Password,
        PoolLimit:    rc.MgoConf.Dial.PoolLimit,
        ReadTimeout:  rc.MgoConf.Dial.ReadTimeout,
        WriteTimeout: rc.MgoConf.Dial.WriteTimeout,
        PoolTimeout:  rc.MgoConf.Dial.PoolTimeout,
        DialServer: func(addr *mgo.ServerAddr) (net.Conn, error) {
            conn, err := tls.Dial("tcp", addr.String(), tlsConfig)

            if err != nil {
                db.logger.Fatal("Dao| ssl mgo conn has err: ", err)
            }

            return conn, err
        },
    }

    for _, addr := range rc.MgoConf.Dial.Addrs {
        db.dialInfo.Addrs = append(db.dialInfo.Addrs, addr)
    }

    // 数据库密码解密
    password, err := encrypt.Dncrypt(db.dialInfo.Password)

    if err != nil {
        return err
    }

    db.dialInfo.Password = password

    return nil
}

// Init 初始化database
func (db *DataBase) Init() {
    rc := db.conf.GetRuntimeConf()

    if rc == nil {
        db.logger.Fatal("Dao| get runtime config failed.")
        util.Exit(-1)
    }

    if err := db.GetDBConf(); err != nil {
        db.logger.Fatal(fmt.Sprintf("Dao| get db conf failed[%s].", err))
        util.Exit(-1)
    }

    if err := db.Dial(); err != nil {
        db.logger.Fatal(fmt.Sprintf("Dao| mgo Init dial failed[%s], exit.", err))
        util.Exit(-1)
    }

    return
}

// Start 启动定时探测数据库连接go rountine
func (db *DataBase) Start() {
    db.Lock()
    defer db.Unlock()

    go db.Run()
}

// ShutDown 停止定时探测数据库连接go rountine
func (db *DataBase) ShutDown() {
    db.Lock()
    defer db.Unlock()

    db.Close()

    if !db.stop {
        db.logger.Info("Dao| mgo refresh go routine shutdown.")
        db.stop = true
        close(db.stopCh)
    }

}

// Run 启动定时探测
func (db *DataBase) Run() {
    rc := db.conf.GetRuntimeConf()
    db.refreshInterval = rc.MgoConf.RefreshInterval
    next := time.After(db.refreshInterval)

    for {
        select {
        case <-next:
            db.Refresh()
            rc := db.conf.GetRuntimeConf()
            db.refreshInterval = rc.MgoConf.RefreshInterval
            next = time.After(db.refreshInterval)
        case <-db.stopCh:
            return
        }
    }
}

// Refresh 定时探测数据库连接是否正常
func (db *DataBase) Refresh() {
    defer func() {
        if err := recover(); err != nil {
            db.logger.Fatal(fmt.Sprintf("Dao| mgo refresh runtime panic[%s].", err))
        }
    }()

    if err := db.session.Ping(); err != nil {
        db.logger.Fatal(fmt.Sprintf("Dao| mgo ping failed[%s], start refresh", err))

        // 尝试重连3次
        for i := 1; i <= 3; i++ {
            db.session.Refresh()

            if err = db.session.Ping(); err != nil {
                db.logger.Fatal(fmt.Sprintf("Dao| mgo refresh times[%d] ping failed[%s].", i, err))
                time.Sleep(1 * time.Second)

                continue
            }

            db.logger.Info(fmt.Sprintf("Dao| mgo refresh times[%d] ping success[%s].", i, err))

            return
        }
    }

    db.logger.Info("Dao| mgo connect to mongodb normal, no need to refresh")

    return
}

// Dial 创建数据库会话
func (db *DataBase) Dial() error {
    db.logger.Info("Dial mongodb")
    // 建立session连接
    var err error
    db.session, err = mgo.DialWithInfo(&db.dialInfo)

    if err != nil {
        str := fmt.Sprintf("Dao|mgo connect Mongo failed[%s]", err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    // 设置数据库模式
    db.session.SetMode(mgo.Strong, true)

    // 创建索引
    // 获取TASKINFO LEVEL的相关信息
    rc := db.conf.GetRuntimeConf()
    taskInfoIndexes := rc.MgoConf.TaskIndexes
    taskInfoIndexesUnique := rc.MgoConf.TaskIndexesUnique
    taskinfoIndexesCompound := rc.MgoConf.TaskIndexesCompound
    levelCount := rc.MgoConf.LevelCount
    levelIndexes := rc.MgoConf.LevelIndexes
    levelIndexesUnique := rc.MgoConf.LevelIndexesUnique
    levelIndexesCompound := rc.MgoConf.LevelIndexesCompound

    // 创建taskinfo索引
    for _, taskInfoIndex := range taskInfoIndexes {
        index := mgo.Index{
            Key:        []string{taskInfoIndex},
            Unique:     false,
            DropDups:   false,
            Background: true,
        }

        err = db.EnsureIndex(structs.TaskInfoTable, index)

        if err != nil {
            str := fmt.Sprintf("Dao|mgo Dial creat taskinfo table normal index failed[%s]", err)
            db.logger.Fatal(str)

            return errors.New(str)
        }
    }

    // 创建taskinfo唯一索引
    for _, taskInfoIndexesUnique := range taskInfoIndexesUnique {
        index := mgo.Index{
            Key:        []string{taskInfoIndexesUnique},
            Unique:     true,
            DropDups:   false,
            Background: true,
        }

        err = db.EnsureIndex(structs.TaskInfoTable, index)

        if err != nil {
            str := fmt.Sprintf("Dao|mgo Dial creat taskinfo table uniq index failed[%s]", err)
            db.logger.Fatal(str)

            return errors.New(str)
        }
    }

    // 创建taskinfo复合索引
    for _, taskinfoIndexCompound := range taskinfoIndexesCompound {
        index := mgo.Index{
            Key:        taskinfoIndexCompound,
            Unique:     false,
            DropDups:   false,
            Background: true,
        }

        err = db.EnsureIndex(structs.TaskInfoTable, index)

        if err != nil {
            str := fmt.Sprintf("Dao|mgo Dial creat taskinfo table compound index failed[%s]", err)
            db.logger.Fatal(str)

            return errors.New(str)
        }
    }

    // 创建level表索引
    for i := 1; i <= levelCount; i++ {
        levelTableName := structs.LevelTable + strconv.Itoa(i)

        for _, levelInfoIndex := range levelIndexes {
            levelIndex := mgo.Index{
                Key:        []string{levelInfoIndex},
                Unique:     false,
                DropDups:   false,
                Background: true,
            }

            err = db.EnsureIndex(levelTableName, levelIndex)

            if err != nil {
                str := fmt.Sprintf("Dao|mgo Dial creat level[%d] table index failed[%s]", i, err)
                db.logger.Fatal(str)

                return errors.New(str)
            }
        }

        //创建唯一索引
        for _, levelInfoIndexUnique := range levelIndexesUnique {
            levelIndex := mgo.Index{
                Key:        []string{levelInfoIndexUnique},
                Unique:     true,
                DropDups:   false,
                Background: true,
            }

            err = db.EnsureIndex(levelTableName, levelIndex)

            if err != nil {
                str := fmt.Sprintf("Dao|mgo Dial creat level[%d] table index failed[%s]", i, err)
                db.logger.Fatal(str)

                return errors.New(str)
            }
        }

        // 创建taskinfo复合索引
        for _, levelIndexCompound := range levelIndexesCompound {
            index := mgo.Index{
                Key:        levelIndexCompound,
                Unique:     false,
                DropDups:   false,
                Background: true,
            }

            err = db.EnsureIndex(levelTableName, index)

            if err != nil {
                str := fmt.Sprintf("Dao|mgo Dial creat level[%d] table compound index failed[%s]", i, err)
                db.logger.Fatal(str)

                return errors.New(str)
            }
        }

    }

    return nil
}

// GetConnection 获取数据库连接COPY
func (db *DataBase) GetConnection() (*mgo.Session, error) {
    return db.session.Clone(), nil
}

// GetCollection 获取collection
func (db *DataBase) GetCollection(session *mgo.Session, collectionName string) (*mgo.Collection, error) {
    c := session.DB(db.dialInfo.Database).C(collectionName)

    return c, nil
}

// EnsureIndex 建立索引
func (db *DataBase) EnsureIndex(collectionName string, index mgo.Index) error {
    c := db.session.DB(db.dialInfo.Database).C(collectionName)
    err := c.EnsureIndex(index)

    if err != nil {
        str := fmt.Sprintf("Dao|mgo creat index failed[%s]", err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    return nil
}

// InsertOneDoc 插入单条数据
func (db *DataBase) InsertOneDoc(collectionName string, doc interface{}) error {
    // 获取连接副本
    session, err := db.GetConnection()

    if err != nil {
        str := fmt.Sprintf("Dao|mgo InsertOneDoc GetConnection failed[%s]", err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    defer session.Close()

    c, _ := db.GetCollection(session, collectionName)
    err = c.Insert(doc)

    if err != nil {
        str := fmt.Sprintf("Dao|mgo InsertOneDoc insert failed[%s]", err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    return nil
}

// BulkInsertDocs 批量插入数据
func (db *DataBase) BulkInsertDocs(collectionName string, doc []interface{}) error {

    if 0 == len(doc) {
        str := fmt.Sprintf("Dao|mgo BulkInsertDocs insert failed, the input is nil")
        db.logger.Fatal(str)

        return errors.New(str)
    }

    // 获取连接副本
    session, err := db.GetConnection()

    if err != nil {
        str := fmt.Sprintf("Dao|mgo BulkInsertDocs GetConnection failed[%s]", err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    defer session.Close()

    c, _ := db.GetCollection(session, collectionName)
    bulk := c.Bulk()
    bulk.Unordered()
    bulk.Insert(doc...)
    _, err = bulk.Run()

    if err != nil {
        str := fmt.Sprintf("Dao|mgo BulkInsertDocs insert failed[%s]", err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    return nil
}

// BulkRemoveDocs 批量删除数据
func (db *DataBase) BulkRemoveDocs(collectionName string, selectors []interface{}) error {
    if 0 == len(selectors) {
        str := fmt.Sprintf("Dao|mgo BulkRemoveDocs failed, the input is nil")
        db.logger.Fatal(str)

        return errors.New(str)
    }

    // 获取连接副本
    session, err := db.GetConnection()

    if err != nil {
        str := fmt.Sprintf("Dao|mgo BulkRemoveDocs GetConnection failed[%s]", err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    defer session.Close()

    c, _ := db.GetCollection(session, collectionName)
    bulk := c.Bulk()
    bulk.Remove(selectors...)
    _, err = bulk.Run()

    if err != nil {
        str := fmt.Sprintf("Dao|mgo BulkRemoveDocs failed[%s]", err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    return nil

}

// FindAll 根据条件查询获取符合条件的所有数据 集合中没有数据，不返回失败
func (db *DataBase) FindAll(selector bson.M, collectionName string, result interface{}) error {
    // 获取连接副本
    session, err := db.GetConnection()

    if err != nil {
        str := fmt.Sprintf("Dao|mgo FindAll GetConnection failed[%s]", err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    defer session.Close()

    c, _ := db.GetCollection(session, collectionName)
    err = c.Find(selector).All(result)

    if err != nil {
        str := fmt.Sprintf("Dao|mgo FindAll  failed[%s]", err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    return nil
}

// FindAllWithLimit 根据条件查询获取符合条件的所有数据 集合中没有数据，不返回失败
func (db *DataBase) FindAllWithLimit(selector bson.M, collectionName string, limit int, result interface{}) error {
    // 获取连接副本
    session, err := db.GetConnection()

    if err != nil {
        str := fmt.Sprintf("Dao|mgo FindAllWithLimit GetConnection failed[%s]", err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    defer session.Close()

    c, _ := db.GetCollection(session, collectionName)
    err = c.Find(selector).Limit(limit).All(result)

    if err != nil {
        str := fmt.Sprintf("Dao|mgo FindAllWithLimit failed[%s]", err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    return nil
}

/*************************************************
// 根据条件查询获取符合条件的所有数据 返回迭代器
func (db *DataBase) Find(selector bson.M, collectionName string) (*mgo.Iter, error) {
    //获取连接副本
    session, err := db.GetConnection()

    if err != nil {
        str := fmt.Sprintf("Dao|mgo Find GetConnection failed[%s]", err)
        db.logger.Fatal(str)

        return nil, errors.New(str)
    }

    defer session.Close()

    c, _ := db.GetCollection(session, collectionName)
    itr := c.Find(selector).Iter()

    return itr, nil
}
***************************************************/

// FindIfExist 根据条件查询，判断是否有对应数据存在
func (db *DataBase) FindIfExist(selector bson.M, collectionName string) (bool, error) {
    // 获取连接副本
    session, err := db.GetConnection()

    if err != nil {
        str := fmt.Sprintf("Dao|mgo FindIfExist GetConnection failed[%s]", err)
        db.logger.Fatal(str)

        return false, errors.New(str)
    }

    defer session.Close()

    c, _ := db.GetCollection(session, collectionName)
    count, err := c.Find(selector).Count()

    if count == 0 {
        return false, nil
    }

    return true, nil
}

// FindOne 根据条件查询获取符合条件1一条数据 集合中没有数据，返回失败
func (db *DataBase) FindOne(selector bson.M, collectionName string, doc interface{}) error {
    // 获取连接副本
    session, err := db.GetConnection()

    if err != nil {
        str := fmt.Sprintf("Dao|mgo FindOne GetConnection failed[%s]", err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    defer session.Close()

    c, _ := db.GetCollection(session, collectionName)
    err = c.Find(selector).One(doc)

    if err != nil {
        str := fmt.Sprintf("Dao|mgo FindOne  failed[%s]", err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    return nil
}

// GetAndRemove 批量1条数据，并将数据删除  集合中没有数据，返回失败
func (db *DataBase) GetAndRemove(collectionName string, result interface{}) error {

    // 获取连接副本
    session, err := db.GetConnection()

    if err != nil {
        str := fmt.Sprintf("Dao|mgo GetAndRemove GetConnection failed[%s]", err)
        db.logger.Fatal(str)

        return err
    }

    defer session.Close()

    c, _ := db.GetCollection(session, collectionName)

    change := mgo.Change{
        Remove:    true,
        ReturnNew: false,
    }

    _, err = c.Find(nil).Apply(change, result)

    if err != nil {
        if err.Error() != DataNotFound {
            str := fmt.Sprintf("Dao|mgo GetAndRemove  failed[%s]", err)
            db.logger.Fatal(str)
        }
    }

    return err
}

// FindAndRemove 批量获取数据，并将数据删除  集合中没有找到则返回失败
func (db *DataBase) FindAndRemove(selector bson.M, collectionName string, results interface{}) error {
    var str string
    exist, _ := db.FindIfExist(selector, collectionName)

    if exist {
        _ = db.FindAll(selector, collectionName, results)
        err := db.RemoveAll(selector, collectionName)

        if err != nil {
            str = fmt.Sprintf("Dao|mgo FindAndRemove  remove failed[%s]", err)
            db.logger.Fatal(str)

            return errors.New(str)
        }

        return nil

    }

    str = fmt.Sprintf("Dao|mgo FindAndRemove  can not find the selector [%v]", selector)

    return errors.New(str)
}

// RemoveAll 批量删除符合查询条件的数据
func (db *DataBase) RemoveAll(selector bson.M, collectionName string) error {
    // 获取连接副本
    session, err := db.GetConnection()

    if err != nil {
        str := fmt.Sprintf("Dao|mgo RemoveAll GetConnection failed[%s]", err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    defer session.Close()

    c, _ := db.GetCollection(session, collectionName)
    _, err = c.RemoveAll(selector)

    if err != nil {
        str := fmt.Sprintf("Dao|mgo RemoveAll failed selector [%v] collectionName[%s] error [%s]", selector, collectionName, err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    return nil
}

// Update 更新一条符合查询条件的数据 找不到数据返回失败
func (db *DataBase) Update(selector bson.M, updateItems bson.M, collectionName string) error {
    // 获取连接副本
    session, err := db.GetConnection()

    if err != nil {
        str := fmt.Sprintf("Dao|mgo Update GetConnection failed[%s]", err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    defer session.Close()

    c, _ := db.GetCollection(session, collectionName)
    err = c.Update(selector, bson.M{"$set": updateItems})

    if err != nil {
        str := fmt.Sprintf("Dao|mgo Update failed selector [%v] collectionName[%s] error [%s]", selector, collectionName, err)
        db.logger.Fatal(str)

        return errors.New(str)
    }

    return nil
}

// GetNextSquence 获取自增序列号 用于taskid生成
func (db *DataBase) GetNextSquence() (int64, error) {
    // 获取连接副本
    session, err := db.GetConnection()

    if err != nil {
        str := fmt.Sprintf("Dao|mgo GetNextSquence GetConnection failed[%s]", err)
        db.logger.Fatal(str)

        return 0, errors.New(str)
    }

    defer session.Close()

    c, _ := db.GetCollection(session, sqTableName)
    cid := "counterid"
    change := mgo.Change{
        Update:    bson.M{"$inc": bson.M{"seq": 1}},
        Upsert:    true,
        ReturnNew: true,
    }
    doc := struct{ Seq int64 }{}

    _, err = c.Find(bson.M{"_id": cid}).Apply(change, &doc)

    if err != nil {
        str := fmt.Sprintf("Dao|mgo GetNextSquence get counter failed:[%s]", err)
        db.logger.Fatal(str)

        return 0, errors.New(str)
    }

    return doc.Seq, nil
}

// FindWithCount 获取collection中满足条件的任务数目
func (db *DataBase) FindWithCount(selector bson.M, collectionName string) (int, error) {
    // 获取连接副本
    session, err := db.GetConnection()

    if err != nil {
        str := fmt.Sprintf("Dao|mgo FindWithCount GetConnection failed[%s]", err)
        db.logger.Fatal(str)

        return 0, errors.New(str)
    }

    defer session.Close()

    c, _ := db.GetCollection(session, collectionName)
    count, err := c.Find(selector).Count()

    if err != nil {
        str := fmt.Sprintf("Dao|mgo selector[%v] table[%s] FindWithCount failed[%s]",
            selector, collectionName, err)
        db.logger.Fatal(str)

        return 0, errors.New(str)
    }

    return count, nil
}
