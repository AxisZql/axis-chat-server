package zlog

import (
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path"
	"runtime"
)

var (
	logger *zap.Logger
)

func init() {
	encoderConfig := zap.NewProductionEncoderConfig()
	// 设置日志记录中时间格式
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	// 日志Encoder 还是JSONEncoder 把日志行格式化成JSON格式的
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	// 644 由于rwx --> rw- r-- r-- 即当前用户读写权限，用户组和其他用户为读权限
	//file, _ := os.OpenFile("/tmp/axis-chat.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 644)
	//fileWriteSyncer := zapcore.AddSync(file)
	// 使用lumberjack进行日志分片
	fileWriteSyncer := getFileLogWriter()

	core := zapcore.NewTee(
		// 同时向控制台和文件写日志，TODO:生产环境应该把控制台写入去掉，日志记录的基本是Debug 及以上，生产环境记得改成Info
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel),
		zapcore.NewCore(encoder, fileWriteSyncer, zapcore.DebugLevel),
	)
	logger = zap.New(core)
}

func getFileLogWriter() (writerSyncer zapcore.WriteSyncer) {
	// Zap 本身不支持日志切割，lumberjack 协助完成切割。
	lumberJckLogger := &lumberjack.Logger{
		Filename:   "/tmp/axis-chat.log",
		MaxSize:    100, //单个文件最大为100M
		MaxAge:     1,   //1天切割一次
		MaxBackups: 60,  //多于60个日志文件后，清理较旧的文件
		Compress:   false,
	}
	return zapcore.AddSync(lumberJckLogger)
}

func Info(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Info(message, fields...)
}

func Debug(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Debug(message, fields...)
}

func Error(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Error(message, fields...)
}

func Warn(message string, fields ...zap.Field) {
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Warn(message, fields...)
}

func getCallerInfoForLog() (callerFields []zap.Field) {
	pc, file, line, ok := runtime.Caller(2) // 回溯两层，拿到写日志的调用方的函数信息
	if !ok {
		return
	}
	funcName := runtime.FuncForPC(pc).Name()
	funcName = path.Base(funcName) //Base函数返回路径的最后一个元素，只保留函数名

	callerFields = append(callerFields, zap.String("func", funcName), zap.String("file", file), zap.Int("line", line))
	return
}
