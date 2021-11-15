package vnc

import (
	"context"
	"fmt"
	"github.com/magnolia-cms/vncrecorder/api"
	"github.com/magnolia-cms/vncrecorder/config"
	"github.com/magnolia-cms/vncrecorder/log"
	"github.com/magnolia-cms/vncrecorder/vnc/recorder"
	"google.golang.org/grpc"
	"math"
	"net"
	_ "net"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type Server struct {
	sync.Mutex
	baseDir    string
	rfiles     map[string]bool
	recordings map[string]*recorder.Recording
	*api.UnimplementedVncRecorderServer
}

func StartServer(done chan bool) error {
	log.Info("-- Starting gRPC server --")

	addr := fmt.Sprintf(":%s", config.Config().GrpcConfig.Port)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Infof("Failed to listen: %v", err)
		return err
	}

	srvOpts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(math.MaxInt64),
		grpc.MaxSendMsgSize(math.MaxInt64),
	}

	grpcServer := grpc.NewServer(srvOpts...)

	s := &Server{
		baseDir:    config.Config().DataDir,
		rfiles:     make(map[string]bool),
		recordings: make(map[string]*recorder.Recording),
	}

	api.RegisterVncRecorderServer(grpcServer, s)

	s.cleanupCron(done)

	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			log.Infof("Failed to serve: %s", err)
		}
	}()

	log.Infof("gRPC listening at %s", addr)

	debug.SetGCPercent(-1)

	go func() {
		for {
			select {
			case <-done:
				log.Infof("Shutting down gRPC")
				grpcServer.GracefulStop()
				return
			case <-time.After(15 * time.Second):
				runtime.GC()
			}
		}
	}()

	return nil
}

func (s *Server) cleanupCron(done chan bool) {
	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(2 * time.Second):
				for fileName := range s.rfiles {
					log.Infof("[CLEANUP] %s", fileName)
					if _, err := os.Stat(fileName); os.IsNotExist(err) {
						delete(s.rfiles, fileName)
					} else if err = os.Remove(fileName); err != nil {
						log.Error(err)
					} else {
						delete(s.rfiles, fileName)
					}
				}
			}
		}
	}()
}

func (s *Server) Start(c context.Context, req *api.VncRequest) (*api.VncResponse, error) {
	t := time.Now()
	fileName := s.filePath(req)
	host := s.host(req)
	log.Infof("-> [START] %s at %s", fileName, host)

	s.stopAndRemoveConn(req)

	recordOptions := recorder.RecordOptions{
		FileName: fileName,
		Host:     host,
	}

	recording, err := recorder.NewRecording(recordOptions)

	if err != nil {
		return &api.VncResponse{
			Status:  api.VncStatus_FAILURE,
			Message: err.Error(),
		}, nil
	}

	recording.Start()

	s.recordings[host] = recording

	log.Infof("<- [START] %s at %s in (%s)", fileName, host, time.Now().Sub(t))

	return &api.VncResponse{
		Status: api.VncStatus_STARTED,
	}, nil
}

func (s *Server) Stop(c context.Context, req *api.VncRequest) (*api.VncResponse, error) {
	t := time.Now()

	host := s.host(req)
	log.Infof("-> [STOP] %s at %s", host, s.filePath(req))

	s.stopAndRemoveConn(req)

	log.Infof("<- [STOP] %s at %s (%s)", s.filePath(req), host, time.Now().Sub(t))

	return &api.VncResponse{
		Status: api.VncStatus_DONE,
	}, nil
}

func (s *Server) Remove(c context.Context, req *api.VncRequest) (*api.VncResponse, error) {
	t := time.Now()

	fileName := s.filePath(req)
	host := s.host(req)
	log.Infof("-> [REMOVE] %s at %s", fileName, host)

	s.stopAndRemoveConn(req)
	s.rfiles[fileName] = true

	log.Infof("<- [REMOVE] %s at %s (%s)", fileName, host, time.Now().Sub(t))

	return &api.VncResponse{
		Status: api.VncStatus_DONE,
	}, nil
}

func (s *Server) stopAndRemoveConn(req *api.VncRequest) {
	s.Lock()
	host := s.host(req)
	if recording, ok := s.recordings[host]; ok {
		recording.Stop()
		delete(s.recordings, host)
	}
	s.Unlock()
}

func (s *Server) host(req *api.VncRequest) string {
	port := config.Config().VncConfig.Port
	if req.Port != nil {
		port = int(*req.Port)
	}
	return fmt.Sprintf("%s:%v", req.Host, port)
}

func (s *Server) filePath(req *api.VncRequest) string {
	var fileName string
	if req.FileName != nil {
		fileName = *req.FileName
	} else {
		fileName = "vnc"
	}
	return fmt.Sprintf("%s/%s.%s", config.Config().DataDir, fileName, s.mediaType(req))
}

func (s *Server) mediaType(req *api.VncRequest) string {
	if req.MediaType == nil {
		return strings.ToLower(api.MediaType_MP4.String())
	}
	return strings.ToLower(req.MediaType.String())
}
