package recorder

import (
	"context"
	vnc "github.com/amitbet/vnc2video"
	"github.com/magnolia-cms/vncrecorder/config"
	"github.com/magnolia-cms/vncrecorder/log"
	"github.com/magnolia-cms/vncrecorder/vnc/encoder"
	"net"
	"time"
)

type RecordOptions struct {
	Host     string
	FileName string
}

type connection struct {
	address   string
	cchServer chan vnc.ServerMessage
	cchClient chan vnc.ClientMessage
	errorCh   chan error
	ccFlags   *vnc.ClientConfig
	conn      *vnc.ClientConn
	screen    *vnc.VncCanvas
	dialer    net.Conn
}

type Recording struct {
	options       RecordOptions
	conn          *connection
	done          chan bool
	restart       bool
	vcodec        *encoder.X264ImageCustomEncoder
	ffmpegStarted chan bool
	ffmpegStopped chan bool
}

func NewRecording(options RecordOptions) (*Recording, error) {
	var (
		err  error
		conn *connection
	)

	vncConfig := config.Config().VncConfig

	conn, err = newConnection(options, vncConfig)
	if err != nil {
		return nil, err
	}

	return &Recording{
		options: options,
		conn:    conn,
		done:    make(chan bool),
		restart: false,
		vcodec: &encoder.X264ImageCustomEncoder{
			Framerate:          vncConfig.FrameRate,
			ConstantRateFactor: vncConfig.Crf,
			VideoFileName:      options.FileName,
		},
		ffmpegStarted: make(chan bool),
		ffmpegStopped: make(chan bool),
	}, nil
}

func (r *Recording) Start() {
	go r.startFFMPEG()
	go r.startEncoding()

	go r.updateVncFrames()

	if !r.restart {
		<-r.done
	}
	log.Infof("[VNC] Recording %s from %s", r.options.FileName, r.conn.address)
}

func (r *Recording) startFFMPEG() {
	//goland:noinspection GoUnhandledErrorResult
	r.vcodec.Run()
	r.ffmpegStarted <- true
}

func (r *Recording) startEncoding() {
	isEncode := true
	for {
		select {
		case <-r.ffmpegStopped:
			log.Infof("[VNC] Stopping encoding %s", r.options.FileName)
			return
		case <-time.After((1000 / time.Duration(r.vcodec.Framerate)) * time.Millisecond):
			if isEncode {
				if ok := r.vcodec.Encode(r.conn.screen.Image); !ok {
					return
				}
			}
		case <-r.conn.errorCh:
			isEncode = false
		}
	}
}

func (r *Recording) updateVncFrames() {
	defer r.conn.close()
	frameBufferReq := 0
	timeStart := time.Now()

	for {
		select {
		case <-r.ffmpegStarted:
			r.done <- true
		case err := <-r.conn.errorCh:
			log.Errorf("Recording for %s failed. Reason: %s", r.options.FileName, err)
			log.Infof("Resetting VNC connection to %s", r.conn.address)
			if err = r.resetVncConnection(); err != nil {
				log.Errorf("Unable to reset VNC connection to %s", r.conn.address)
			}
		case msg := <-r.conn.cchClient:
			log.Infof("client message { messageType: %v, message: %v } received", msg.Type(), msg)
		case msg := <-r.conn.cchServer:
			if msg.Type() == vnc.FramebufferUpdateMsgType {
				secsPassed := time.Now().Sub(timeStart).Seconds()
				frameBufferReq++
				reqPerSec := float64(frameBufferReq) / secsPassed
				log.Debugf("[VNC] framebuffer update { reqs: %d, seconds: %v, rate: %v }", frameBufferReq, secsPassed, reqPerSec)
				reqMsg := vnc.FramebufferUpdateRequest{Inc: 1, X: 0, Y: 0, Width: r.conn.conn.Width(), Height: r.conn.conn.Height()}
				//goland:noinspection GoUnhandledErrorResult
				reqMsg.Write(r.conn.conn)
			}
		case <-r.done:
			r.vcodec.Close()
			r.done <- true
			r.ffmpegStopped <- true
			return
		}
	}
}

func (r *Recording) resetVncConnection() error {
	var (
		err  error
		conn *connection
	)

	r.conn.close()
	vncConfig := config.Config().VncConfig
	conn, err = newConnection(r.options, vncConfig)
	if err != nil {
		return err
	}

	r.conn = conn

	return nil
}

func (r *Recording) Stop() {
	r.done <- true
	<-r.done
}

func newConnection(options RecordOptions, vncConfig config.VncConfig) (*connection, error) {
	t := time.Now()

	dialer, err := net.DialTimeout("tcp", options.Host, 5*time.Second)
	if err != nil {
		log.Errorf("connection to VNC host failed. Reason %s", err)
		return nil, err
	}

	// Negotiate connection with the server.
	cchServer := make(chan vnc.ServerMessage)
	cchClient := make(chan vnc.ClientMessage)
	errorCh := make(chan error)

	var secHandlers []vnc.SecurityHandler
	if len(vncConfig.Password) == 0 {
		secHandlers = []vnc.SecurityHandler{
			&vnc.ClientAuthNone{},
		}
	} else {
		secHandlers = []vnc.SecurityHandler{
			&vnc.ClientAuthVNC{Password: []byte(vncConfig.Password)},
		}
	}

	ccflags := &vnc.ClientConfig{
		SecurityHandlers: secHandlers,
		DrawCursor:       true,
		PixelFormat:      vnc.PixelFormat32bit,
		ClientMessageCh:  cchClient,
		ServerMessageCh:  cchServer,
		Messages:         vnc.DefaultServerMessages,
		Encodings: []vnc.Encoding{
			&vnc.RawEncoding{},
			&vnc.TightEncoding{},
			&vnc.HextileEncoding{},
			&vnc.ZRLEEncoding{},
			&vnc.CopyRectEncoding{},
			&vnc.CursorPseudoEncoding{},
			&vnc.CursorPosPseudoEncoding{},
			&vnc.ZLibEncoding{},
			&vnc.RREEncoding{},
			&vnc.DesktopNamePseudoEncoding{},
			&vnc.DesktopSizePseudoEncoding{},
			&vnc.AtenHermon{},
		},
		ErrorCh: errorCh,
	}

	vncConnection, err := vnc.Connect(context.Background(), dialer, ccflags)

	if err != nil {
		log.Errorf("connection negotiation to VNC host failed. Reason %s", err)
		return nil, err
	}

	log.Infof("[VNC] Connected %s in %s.", options.Host, time.Now().Sub(t))

	//goland:noinspection GoUnhandledErrorResult
	vncConnection.SetEncodings([]vnc.EncodingType{
		vnc.EncCursorPseudo,
		vnc.EncPointerPosPseudo,
		vnc.EncCopyRect,
		vnc.EncTight,
		vnc.EncZRLE,
		vnc.EncHextile,
		vnc.EncZlib,
		vnc.EncRRE,
		vnc.EncRaw,
		vnc.EncDesktopNamePseudo,
		vnc.EncDesktopSizePseudo,
		vnc.EncAtenHermon,
	})

	for _, enc := range ccflags.Encodings {
		myRenderer, ok := enc.(vnc.Renderer)

		if ok {
			myRenderer.SetTargetImage(vncConnection.Canvas)
		}
	}

	return &connection{
		cchClient: cchClient,
		cchServer: cchServer,
		errorCh:   errorCh,
		ccFlags:   ccflags,
		conn:      vncConnection,
		dialer:    dialer,
		address:   options.Host,
		screen:    vncConnection.Canvas,
	}, nil
}

func (c connection) close() {
	//goland:noinspection GoUnhandledErrorResult
	c.dialer.Close()
	//goland:noinspection GoUnhandledErrorResult
	c.conn.Close()
	log.Infof("[VNC] Disconnected from %s", c.address)
}
