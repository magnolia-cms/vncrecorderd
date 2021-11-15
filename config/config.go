package config

import (
	"github.com/magnolia-cms/vncrecorder/log"
	"os"
	"strconv"
)

type config struct {
	GrpcConfig
	VncConfig
}

type GrpcConfig struct {
	Port string
}

type VncConfig struct {
	FrameRate int
	Crf       int
	DataDir string
	Password string
	Host string
	Port int
}

var cfg *config

var Config = func() *config {
	if cfg == nil {
		cfg = &config{}

		if cfg.GrpcConfig.Port = os.Getenv("GRPC_PORT"); len(cfg.GrpcConfig.Port) == 0 {
			cfg.GrpcConfig.Port = "3000"
		}

		if port := os.Getenv("VNC_PORT"); len(port) == 0 {
			cfg.VncConfig.Port = 5900
		} else {
			port, err := strconv.Atoi(port)
			if err != nil {
				log.Error(err)
			} else {
				cfg.VncConfig.Port = port
			}
		}

		if cfg.VncConfig.Password = os.Getenv("VNC_PASSWORD"); len(cfg.VncConfig.Password) == 0 {
			cfg.VncConfig.Password = "secret"
		}

		cfg.VncConfig.Host = os.Getenv("VNC_HOST")

		if frameRate := os.Getenv("VNC_FRAME_RATE"); len(frameRate) == 0 {
			cfg.VncConfig.FrameRate = 60
		} else {
			frameRate, _ := strconv.Atoi(frameRate)
			cfg.VncConfig.FrameRate = frameRate
		}

		if crf := os.Getenv("VNC_CONSTANT_RATE_FACTOR"); len(crf) == 0 {
			cfg.VncConfig.Crf = 0
		} else {
			crf, _ := strconv.Atoi(crf)
			cfg.VncConfig.Crf = crf
		}

		if cfg.VncConfig.DataDir = os.Getenv("VNC_RECORDINGS_DIR"); len(cfg.VncConfig.DataDir) == 0 {
			cfg.VncConfig.DataDir = "/recordings"
		}
	}
	return cfg
}