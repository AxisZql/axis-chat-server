package config

import (
	"github.com/spf13/viper"
	"os"
	"runtime"
	"strings"
	"sync"
)

/**
*Author: AxisZql
*Date: 2022-5-31
*DESC: the configuration parse for this project
 */

const (
	SuccessReplyCode   = 0
	FailReplyCode      = 1
	RedisBaseValidTime = 86400
)

type Config struct {
	Common struct {
		Db struct {
			Address   string `mapstructure:"address"`
			Username  string `mapstructure:"username"`
			Password  string `mapstructure:"password"`
			DbName    string `mapstructure:"dbName"`
			InitModel bool   `mapstructure:"dbName"`
		} `mapstructure:"db"`

		Redis struct {
			Address  string `mapstructure:"address"`
			Password string `mapstructure:"password"`
			Db       int    `mapstructure:"db"`
		} `mapstructure:"redis"`

		Kafka struct {
			Address  string `mapstructure:"address"`
			Username string `mapstructure:"username"`
			Password string `mapstructure:"password"`
		}

		Etcd struct {
			Address           string `mapstructure:"address"`
			BasePath          string `mapstructure:"bsePath"`
			ServerPathLogic   string `mapstructure:"serverPathLogic"`
			ServerPathConnect string `mapstructure:"serverPathConnect"`
			Username          string `mapstructure:"username"`
			Password          string `mapstructure:"password"`
			ConnectionTimeout int    `mapstructure:"connectionTimeout"`
		} `mapstructure:"etcd"`
	}
	Api struct {
		Api struct {
			Host string `mapstructure:"host"`
			Port int    `mapstructure:"port"`
		} `mapstructure:"api"`
	}

	LogicRpc struct {
		Logic struct {
			Host       string `mapstructure:"host"`
			RpcAddress string `mapstructure:"rpcAddress"`
			CerPath    string `mapstructure:"cerPath"`
			KeyPath    string `mapstructure:"keyPath"`
		} `mapstructure:"logic"`
	}
	ConnectRpc struct {
		ConnectBucket struct {
			CpuNum int `mapstructure:"cpuNum"`
		} `mapstructure:"connect-bucket"`
		ConnectWebsocket struct {
			Host       string `mapstructure:"host"`
			Bind       string `mapstructure:"bind"` //websocket服务监听的端口
			RpcAddress string `mapstructure:"rpcAddress"`
			CerPath    string `mapstructure:"cerPath"`
			KeyPath    string `mapstructure:"keyPath"`
		}
	}
}

var (
	once sync.Once
	conf Config
)

func GetConfig() *Config {
	return &conf
}

func InitConfig() {
	// once.Do单例模式
	once.Do(func() {
		env := GetGinRunMode()
		configFilePath := GetCurrentDirPath() + "/" + env + "/"
		viper.SetConfigType("toml")
		viper.SetConfigName("api")
		viper.AddConfigPath(configFilePath)
		err := viper.ReadInConfig()
		if err != nil {
			panic(err)
		}
		viper.SetConfigName("common")
		err = viper.MergeInConfig()
		if err != nil {
			panic(err)
		}
		viper.SetConfigName("task")
		err = viper.MergeInConfig()
		if err != nil {
			panic(err)
		}
		viper.SetConfigName("connect")
		err = viper.MergeInConfig()
		if err != nil {
			panic(err)
		}
		viper.SetConfigName("logic")
		err = viper.MergeInConfig()
		if err != nil {
			panic(err)
		}
		viper.SetConfigName("task")
		err = viper.MergeInConfig()
		if err != nil {
			panic(err)
		}
		_ = viper.Unmarshal(&conf.Api)
		_ = viper.Unmarshal(&conf.Common)
		_ = viper.Unmarshal(&conf.ConnectRpc)
		_ = viper.Unmarshal(&conf.LogicRpc)
		//_ = viper.Unmarshal(&conf.Task)

	})
}

// GetCurrentDirPath 获取config的绝对路径
func GetCurrentDirPath() string {
	_, filename, _, _ := runtime.Caller(1)
	aPath := strings.Split(filename, "/")
	// 将../config/config.go 转换为../config/
	dir := strings.Join(aPath[:len(aPath)-1], "/")
	return dir
}

func GetRunMode() string {
	env := os.Getenv("RUN_MODE")
	if env == "" {
		return "dev"
	}
	return env
}
func GetGinRunMode() string {
	env := GetRunMode()
	if env == "dev" {
		return "debug"
	} else if env == "test" {
		return "debug"
	} else if env == "prod" {
		return "release"
	}
	return "release"
}
