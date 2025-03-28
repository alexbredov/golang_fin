package loggercli

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
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"cli_full.log"},
		ErrorOutputPaths: []string{"cli_error.log"},
	}
	logWrapper.logger = zap.Must(logWrapper.config.Build()).Sugar()
	return &logWrapper, nil
}
func (logWrapper LogWrapper) Info(msg string) {
	logWrapper.logger.Info(msg)
}
func (logWrapper LogWrapper) Error(msg string) {
	logWrapper.logger.Error(msg)
}
func (logWrapper LogWrapper) Fatal(msg string) {
	logWrapper.logger.Fatal(msg)
}
