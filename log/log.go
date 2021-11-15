package log

import (
	log2 "github.com/rish1988/go-log"
	log2config "github.com/rish1988/go-log/config"
	"os"
)

var (
	log    *log2.Logger
	Info   = logger().Info
	Infof  = logger().Infof
	Error  = logger().Error
	Errorf = logger().Errorf
	Warn   = logger().Warn
	Warnf  = logger().Warnf
	Debug  = logger().Debug
	Debugf = logger().Debugf
)

func logger() *log2.Logger {
	if log == nil {
		opts := log2config.LogOptions{
			ColorOptions: log2config.ColorOptions{
				TimeStampColorOptions: log2config.TimeStampColorOptions{
					TimeStamp: true,
				},
			},
			Debug: false,
		}

		if fileName := os.Getenv("LOG_FILE"); len(fileName) != 0 {
			if file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600); err != nil {
				log = log2.New(log2.NewFdWriters(os.Stderr), opts)
			} else {
				log = log2.New(log2.NewFdWriters(os.Stderr, file), opts)
			}
		} else {
			log = log2.New(log2.NewFdWriters(os.Stderr), opts)
		}
	}
	return log
}
