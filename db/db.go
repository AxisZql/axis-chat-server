package db

import (
	"axisChat/config"
	"axisChat/utils"
	"axisChat/utils/zlog"
	"fmt"
	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

/**
*Author: AxisZql
*Date: 2022-5-31
*DESC: about DB operation. Only logic layer can use db
 */

var (
	once          sync.Once
	db            *gorm.DB
	vGroupMessage = `
    create view v_group_message as
    select mg.*, t_user.username as 'from_username', t_user.avatar
    from (select m.*, t_group.group_name
          from (select id,
                       from_a as 'userid',
                       to_b   as 'group_id',
                       content,
                       message_type,
                       create_at,
                       snow_id,
                       belong
                from t_message
                where type = 'group'
                order by snow_id DESC) m
                   join t_group on m.group_id = t_group.id) mg
             join t_user on t_user.id = mg.userid;`
	vFriendMessage = `
    create view v_friend_message as
    select mf.*, t_user.username as 'from_username', t_user.avatar
    from (select m.*, friend.username as 'friend_name'
          from (select id,
                       from_a as 'userid',
                       to_b   as 'friend_id',
                       content,
                       message_type,
                       create_at,
                       snow_id,
                       belong
                from t_message
                where type = 'friend'
                order by snow_id DESC) m
                   join t_user as friend on m.friend_id = friend.id) mf
             join t_user on t_user.id = mf.userid;`
)

// InitDb 初始化客户端连接配置
func InitDb() {
	once.Do(func() {
		conf := config.GetConfig()
		host, port, err := utils.ParseAddress(conf.Common.Db.Address)
		if err != nil {
			panic(err)
		}
		dbInfo := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			conf.Common.Db.Username, conf.Common.Db.Password, host, port, conf.Common.Db.DbName)
		db, err = gorm.Open(mysql.Open(dbInfo), &gorm.Config{
			Logger: logger.New(
				log.New(os.Stdout, "\r\n", log.LstdFlags),
				logger.Config{
					SlowThreshold: time.Second,
					LogLevel:      logger.Info,
					Colorful:      true,
				},
			),
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true, //使用单数表名
			},
		})
		if err != nil {
			panic(err)
		}
		zlog.Info(fmt.Sprintf("数据库:%s:%d;%s", host, port, conf.Common.Db.DbName))
		sqlDB, err := db.DB()
		if err != nil {
			panic(err)
		}
		//设置连接池中最大大闲连接数
		sqlDB.SetMaxIdleConns(10)
		//设置数据库大最大连接数
		sqlDB.SetMaxOpenConns(5)
		//设置连接大最大可复用时间
		sqlDB.SetConnMaxLifetime(time.Hour)

		if conf.Common.Db.InitModel {
			t := time.Now()
			modelsInit()
			zlog.Info(fmt.Sprintf("inti models in:%v", time.Since(t)))
		}
	})

}

func modelsInit() {
	zlog.Info("models initializing...")
	e1 := db.AutoMigrate(&TUser{}, &TGroup{}, &TMessage{}, &TRelation{})
	if e1 != nil {
		err := errors.Wrap(e1, "初始化表失败")
		panic(err)
	}
	e2 := db.Exec("drop view v_group_message;")
	if e2.Error != nil && !strings.Contains(e2.Error.Error(), "Unknown table") {
		panic(e2)
	}
	e2 = db.Exec(vGroupMessage)
	if e2.Error != nil {
		panic(e2)
	}
	e3 := db.Exec("drop view v_friend_message;")
	if e3.Error != nil && !strings.Contains(e3.Error.Error(), "Unknown table") {
		panic(e3)
	}
	e3 = db.Exec(vFriendMessage)
	if e3.Error != nil {
		panic(e3)
	}
}

func GetDb() *gorm.DB {
	return db
}
