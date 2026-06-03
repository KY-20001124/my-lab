package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func InitLogger() {
	// 配置日志编码器
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.LowercaseLevelEncoder,
		EncodeTime:    zapcore.ISO8601TimeEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
	}

	// 设置日志级别（Info 级别及以上输出）
	atomicLevel := zap.NewAtomicLevelAt(zap.InfoLevel)

	// 输出到控制台
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout)),
		atomicLevel,
	)

	// 初始化全局 Logger
	Log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	defer Log.Sync() // 确保日志 buffer 刷新
}
