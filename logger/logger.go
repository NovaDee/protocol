package logger

import (
	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
	"sync"
	"time"
)

var (
	discardLogger        = logr.Discard()
	defaultLogger Logger = LogRLogger(discardLogger)
	pkgLogger     Logger = LogRLogger(discardLogger)
)

// InitLogConfig 初始化已定义日志
func InitLogConfig(conf Config, system string) {
	l, err := NewZapLogger(&conf)
	if err == nil {
		SetLogger(l, system)
	}
}

type Logger interface {
	Debugw(msg string, keysAndValues ...interface{})
	Infow(msg string, keysAndValues ...interface{})
	Warnw(msg string, err error, keysAndValues ...interface{})
	Errorw(msg string, err error, keysAndValues ...interface{})
	WithValues(keysAndValues ...interface{}) Logger
	WithName(name string) Logger
	// WithComponent creates a new logger with name as "<name>.<component>", and uses a log level as specified
	WithComponent(component string) Logger
	WithCallDepth(depth int) Logger
	WithItemSampler() Logger
	// WithoutSampler returns the original logger without sampling
	WithoutSampler() Logger
}

func (l LogRLogger) toLogr() logr.Logger {
	if logr.Logger(l).GetSink() == nil {
		return discardLogger
	}
	return logr.Logger(l)
}

func (l LogRLogger) Debugw(msg string, keysAndValues ...interface{}) {
	l.toLogr().V(1).Info(msg, keysAndValues...)
}

func (l LogRLogger) Infow(msg string, keysAndValues ...interface{}) {
	l.toLogr().Info(msg, keysAndValues...)
}

func (l LogRLogger) Warnw(msg string, err error, keysAndValues ...interface{}) {
	if err != nil {
		keysAndValues = append(keysAndValues, "error", err)
	}
	l.toLogr().Info(msg, keysAndValues...)
}

func (l LogRLogger) Errorw(msg string, err error, keysAndValues ...interface{}) {
	l.toLogr().Error(err, msg, keysAndValues...)
}

func (l LogRLogger) WithValues(keysAndValues ...interface{}) Logger {
	return LogRLogger(l.toLogr().WithValues(keysAndValues...))
}

func (l LogRLogger) WithName(name string) Logger {
	return LogRLogger(l.toLogr().WithName(name))
}

func (l LogRLogger) WithComponent(component string) Logger {
	return LogRLogger(l.toLogr().WithName(component))
}

func (l LogRLogger) WithCallDepth(depth int) Logger {
	return LogRLogger(l.toLogr().WithCallDepth(depth))
}

func (l LogRLogger) WithItemSampler() Logger {
	return l
}

func (l LogRLogger) WithoutSampler() Logger {
	return l
}

type ZapLogger struct {
	zap *zap.SugaredLogger
	// store original logger without sampling to avoid multiple samplers
	unsampled *zap.SugaredLogger
	component string
	// use a nested field as pointer so that all loggers share the same sharedConfig
	sharedConfig   *sharedConfig
	level          zap.AtomicLevel
	SampleDuration time.Duration
	SampleInitial  int
	SampleInterval int
}

func NewZapLogger(conf *Config) (*ZapLogger, error) {
	sc := newSharedConfig(conf)
	zaplog := &ZapLogger{
		sharedConfig:   sc,
		level:          sc.level,
		SampleDuration: time.Duration(conf.ItemSampleSeconds) * time.Second,
		SampleInitial:  conf.ItemSampleInitial,
		SampleInterval: conf.ItemSampleInterval,
	}
	zc := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapcore.DebugLevel),
		Development:      false,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	if conf.JSON {
		zc.Encoding = "json"
		zc.EncoderConfig = zap.NewProductionEncoderConfig()
		zc.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	build, err := zc.Build()
	if err != nil {
		return nil, err
	}
	zaplog.unsampled = build.Sugar()

	if conf.Sample {
		samplingConf := zap.SamplingConfig{
			Initial:    conf.ItemSampleInitial,
			Thereafter: conf.SampleInterval,
		}

		// sane defaults
		if samplingConf.Initial == 0 {
			samplingConf.Initial = 20
		}
		if samplingConf.Thereafter == 0 {
			samplingConf.Thereafter = 100
		}

		zaplog.zap = build.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewSamplerWithOptions(
				core,
				time.Second,
				samplingConf.Initial,
				samplingConf.Thereafter,
			)
		})).Sugar()
	} else {
		zaplog.zap = zaplog.unsampled
	}
	return zaplog, nil
}

type sharedConfig struct {
	level           zap.AtomicLevel
	lc              sync.Mutex
	componentLevels map[string]zap.AtomicLevel
	config          *Config
}

func newSharedConfig(conf *Config) *sharedConfig {
	return &sharedConfig{
		level:           zap.NewAtomicLevelAt(ParseZapLevel(conf.Level)),
		config:          conf,
		componentLevels: make(map[string]zap.AtomicLevel),
	}
	//conf.AddUpdateObserver(sc.)
}

// GetLogger returns the logger that was set with SetLogger with an extra depth of 1
func GetLogger() Logger {
	return defaultLogger
}

// SetLogger lets you use a custom logger. Pass in a logr.Logger with default depth
func SetLogger(l Logger, name string) {
	defaultLogger = l.WithCallDepth(1).WithName(name)
	// pkg wrapper needs to drop two levels of depth
	pkgLogger = l.WithCallDepth(2).WithName(name)
}

func Debugw(msg string, keysAndValues ...interface{}) {
	pkgLogger.Debugw(msg, keysAndValues...)
}

func Infow(msg string, keysAndValues ...interface{}) {
	pkgLogger.Infow(msg, keysAndValues...)
}

func Warnw(msg string, err error, keysAndValues ...interface{}) {
	pkgLogger.Warnw(msg, err, keysAndValues...)
}

func Errorw(msg string, err error, keysAndValues ...interface{}) {
	pkgLogger.Errorw(msg, err, keysAndValues...)
}

func ParseZapLevel(level string) zapcore.Level {
	lvl := zapcore.InfoLevel
	if level != "" {
		_ = lvl.UnmarshalText([]byte(level))
	}
	return lvl
}

func (l *ZapLogger) ToZap() *zap.SugaredLogger {
	return l.zap
}

type LogRLogger logr.Logger

func (l *ZapLogger) isEnabled(level zapcore.Level) bool {
	return level >= l.level.Level()
}

func (l *ZapLogger) Debugw(msg string, keysAndValues ...interface{}) {
	if !l.isEnabled(zapcore.DebugLevel) {
		return
	}
	l.zap.Debugw(msg, keysAndValues...)
}

func (l *ZapLogger) Infow(msg string, keysAndValues ...interface{}) {
	if !l.isEnabled(zapcore.InfoLevel) {
		return
	}
	l.zap.Infow(msg, keysAndValues...)
}

func (l *ZapLogger) Warnw(msg string, err error, keysAndValues ...interface{}) {
	if !l.isEnabled(zapcore.WarnLevel) {
		return
	}
	if err != nil {
		keysAndValues = append(keysAndValues, "error", err)
	}
	l.zap.Warnw(msg, keysAndValues...)
}

func (l *ZapLogger) Errorw(msg string, err error, keysAndValues ...interface{}) {
	if !l.isEnabled(zapcore.ErrorLevel) {
		return
	}
	if err != nil {
		keysAndValues = append(keysAndValues, "error", err)
	}
	l.zap.Errorw(msg, keysAndValues...)
}

func (l *ZapLogger) WithValues(keysAndValues ...interface{}) Logger {
	dup := *l
	dup.zap = l.zap.With(keysAndValues...)
	// mirror unsampled logger too
	if l.unsampled == l.zap {
		dup.unsampled = dup.zap
	} else {
		dup.unsampled = l.unsampled.With(keysAndValues...)
	}
	return &dup
}

func (l *ZapLogger) WithName(name string) Logger {
	dup := *l
	dup.zap = l.zap.Named(name)
	if l.unsampled == l.zap {
		dup.unsampled = dup.zap
	} else {
		dup.unsampled = l.unsampled.Named(name)
	}
	return &dup
}

func (l *ZapLogger) WithComponent(component string) Logger {
	// zap automatically appends .<name> to the logger name
	dup := l.WithName(component).(*ZapLogger)
	if dup.component == "" {
		dup.component = component
	} else {
		dup.component = dup.component + "." + component
	}
	dup.level = dup.sharedConfig.setEffectiveLevel(dup.component)
	return dup
}

func (l *ZapLogger) WithCallDepth(depth int) Logger {
	dup := *l
	dup.zap = l.zap.WithOptions(zap.AddCallerSkip(depth))
	if l.unsampled == l.zap {
		dup.unsampled = dup.zap
	} else {
		dup.unsampled = l.unsampled.WithOptions(zap.AddCallerSkip(depth))
	}
	return &dup
}

func (l *ZapLogger) WithItemSampler() Logger {
	if l.SampleDuration == 0 {
		return l
	}
	dup := *l
	dup.zap = l.unsampled.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewSamplerWithOptions(
			core,
			l.SampleDuration,
			l.SampleInitial,
			l.SampleInterval,
		)
	}))
	return &dup
}

func (l *ZapLogger) WithoutSampler() Logger {
	if l.unsampled == l.zap {
		return l
	}
	dup := *l
	dup.zap = l.unsampled
	return &dup
}

// 动态更新日志等级，后续使用
func (cfg *sharedConfig) onConfigUpdate(conf *Config) error {
	// 设置最新的日志等级
	cfg.level.SetLevel(ParseZapLevel(conf.Level))
	cfg.lc.Lock()
	defer cfg.lc.Unlock()
	cfg.config = conf
	for component, atomicLevel := range cfg.componentLevels {
		updateLevel := cfg.level.Level()
		parts := strings.Split(component, ".")
	confSearch:
		for len(parts) > 0 {
			search := strings.Join(parts, ".")
			if compLevel, ok := conf.ComponentLevels[search]; ok {
				updateLevel = ParseZapLevel(compLevel)
				break confSearch
			}
			parts = parts[:len(parts)-1]
		}
		atomicLevel.SetLevel(updateLevel)
	}
	return nil
}

// 动态更新日志等级，后续使用
func (c *sharedConfig) setEffectiveLevel(component string) zap.AtomicLevel {
	c.lc.Lock()
	defer c.lc.Unlock()
	if compLevel, ok := c.componentLevels[component]; ok {
		return compLevel
	}

	// search up the hierarchy to find the first level that is set
	atomicLevel := zap.NewAtomicLevelAt(c.level.Level())
	c.componentLevels[component] = atomicLevel
	parts := strings.Split(component, ".")
	for len(parts) > 0 {
		search := strings.Join(parts, ".")
		if compLevel, ok := c.config.ComponentLevels[search]; ok {
			atomicLevel.SetLevel(ParseZapLevel(compLevel))
			return atomicLevel
		}
		parts = parts[:len(parts)-1]
	}
	return atomicLevel
}
