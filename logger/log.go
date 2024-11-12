package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Level = zapcore.Level

const (
	InfoLevel   Level = zap.InfoLevel   // 0, default level
	WarnLevel   Level = zap.WarnLevel   // 1
	ErrorLevel  Level = zap.ErrorLevel  // 2
	DPanicLevel Level = zap.DPanicLevel // 3, used in development log
	// PanicLevel logs a message, then panics
	PanicLevel Level = zap.PanicLevel // 4
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel Level = zap.FatalLevel // 5
	DebugLevel Level = zap.DebugLevel // -1
)

type Field = zap.Field

func (l *Logger) Debug(msg string, arg ...interface{}) {
	if len(arg) > 0 {
		l.sugarZapInstance.Debugf(msg, arg...)
	} else {
		l.sugarZapInstance.Debug(msg)
	}
}

func (l *Logger) Info(msg string, arg ...interface{}) {
	if len(arg) > 0 {
		l.sugarZapInstance.Infof(msg, arg...)
	} else {
		l.sugarZapInstance.Info(msg)
	}
}

func (l *Logger) Warn(msg string, arg ...interface{}) {
	if len(arg) > 0 {
		l.sugarZapInstance.Warnf(msg, arg...)
	} else {
		l.sugarZapInstance.Warn(msg)
	}
}

func (l *Logger) Error(msg string, arg ...interface{}) {
	if len(arg) > 0 {
		l.sugarZapInstance.Errorf(msg, arg...)
	} else {
		l.sugarZapInstance.Error(msg)
	}
}
func (l *Logger) DPanic(msg string, arg ...interface{}) {
	if len(arg) > 0 {
		l.sugarZapInstance.DPanicf(msg, arg...)
	} else {
		l.sugarZapInstance.DPanic(msg)
	}
}
func (l *Logger) Panic(msg string, arg ...interface{}) {
	if len(arg) > 0 {
		l.sugarZapInstance.Panicf(msg, arg...)
	} else {
		l.sugarZapInstance.Panic(msg)
	}
}
func (l *Logger) Fatal(msg string, arg ...interface{}) {
	if len(arg) > 0 {
		l.sugarZapInstance.Fatalf(msg, arg...)
	} else {
		l.sugarZapInstance.Fatal(msg)
	}
}

func (l *Logger) HDebug(msg string, fields ...Field) {
	l.zapInstance.Debug(msg, fields...)
}

func (l *Logger) HInfo(msg string, fields ...Field) {
	l.zapInstance.Info(msg, fields...)
}

func (l *Logger) HWarn(msg string, fields ...Field) {
	l.zapInstance.Warn(msg, fields...)
}

func (l *Logger) HError(msg string, fields ...Field) {
	l.zapInstance.Error(msg, fields...)
}
func (l *Logger) HDPanic(msg string, fields ...Field) {
	l.zapInstance.DPanic(msg, fields...)
}
func (l *Logger) HPanic(msg string, fields ...Field) {
	l.zapInstance.Panic(msg, fields...)
}
func (l *Logger) HFatal(msg string, fields ...Field) {
	l.zapInstance.Fatal(msg, fields...)
}

// function variables for all field types
// in github.com/uber-go/zap/field.go

var (
	FieldSkip       = zap.Skip
	FieldBinary     = zap.Binary
	FieldBool       = zap.Bool
	FieldBoolp      = zap.Boolp
	FieldByteString = zap.ByteString
	FieldFloat64    = zap.Float64
	FieldFloat64p   = zap.Float64p
	FieldFloat32    = zap.Float32
	FieldFloat32p   = zap.Float32p
	FieldDurationp  = zap.Durationp
	FieldDuration   = zap.Duration
	FieldString     = zap.String
	FieldTime       = zap.Time
	FieldInt        = zap.Int

	FieldAny = zap.Any
)

type Logger struct {
	zapInstance      *zap.Logger // zap ensure that zap.Logger is safe for concurrent use
	sugarZapInstance *zap.SugaredLogger
	level            Level
}

func (l *Logger) Sync() error {
	err := l.zapInstance.Sync()
	if err != nil {
		return err
	}
	return l.sugarZapInstance.Sync()
}

func GetZapSugar() *zap.SugaredLogger {
	return defaultLogger.sugarZapInstance
}

func GetZap() *zap.Logger {
	return defaultLogger.zapInstance
}
