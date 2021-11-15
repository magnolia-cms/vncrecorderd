package main

import (
	"github.com/magnolia-cms/vncrecorder/log"
	"github.com/magnolia-cms/vncrecorder/vnc"
	"os"
	"time"
)

func main() {
	t := time.Now()

	done := make(chan bool)

	if err := vnc.StartServer(done); err != nil {
		log.Error(err)
		os.Exit(0)
	} else {
		log.Infof("Server started in %s", time.Now().Sub(t))
	}

	<-done
}
