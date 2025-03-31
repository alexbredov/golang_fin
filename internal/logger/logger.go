package logger

import "go.uber.org/zap"

type LogWrapper struct {
	config zap.Config
	logger *zap.SugaredLogger
}

func New(level string) (*LogWrapper, error) {
	logWrapper := LogWrapper{}
	zlevel, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, err
	}
	logWrapper.config = zap.Config{
		Level:            zlevel,
		DisableCaller:    true,
		Development:      true,
		Encoding:         "console",
		OutputPaths:      []string{"stdout", "full_log.log"},
		ErrorOutputPaths: []string{"stderr", "error_log.log"},
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
	}
	logWrapper.logger = zap.Must(logWrapper.config.Build()).Sugar()
	return &logWrapper, nil
}

func (log LogWrapper) GetZapLogger() *zap.SugaredLogger {
	return log.logger
}

func (log LogWrapper) Info(msg string) {
	log.logger.Info(msg)
}

func (log LogWrapper) Warning(msg string) {
	log.logger.Warn(msg)
}

func (log LogWrapper) Error(msg string) {
	log.logger.Error(msg)
}

func (log LogWrapper) Fatal(msg string) {
	log.logger.Fatal(msg)
}
