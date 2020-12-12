package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gosm/pkg/live"
	"gosm/pkg/log"
	"gosm/pkg/protocol/httpflv"
	"gosm/pkg/protocol/rtmp"
	"gosm/pkg/utils"
)

const (
	_LogPrefiex = "[GoSM]"
	_PortRTMP   = "1935"
	_PortFlv    = "8088"
	_Version    = "0.0.1"
)

func main() {
	fmt.Printf(`
      _____       _____ __  __ 
     / ____|     / ____|  \/  |
    | |  __  ___| (___ | \  / |
    | | |_ |/ _ \\___ \| |\/| |
    | |__| | (_) |___) | |  | |
     \_____|\___/_____/|_|  |_|    version: %s`, _Version+"\n\n")

	// log.SetLevel(log.LevelDebug)
	log.SetPrefix(_LogPrefiex)

	// id worker
	idworker, err := utils.NewSnowflake(1, 1)

	// living room managerment
	roomMgmt, err := live.NewRoomMgmt(idworker)

	// rtmp server
	rtmpServer, rtmpCloseFunc, err := rtmp.NewServer("tcp", ":"+_PortRTMP, roomMgmt)
	if err != nil {
		log.Fatal("RTMP Server Start Faild:%v", err)
	}
	rtmpServer.Start()

	// http-flv server
	flvServer, flvCloseFunc, err := httpflv.NewServer("tcp", ":"+_PortFlv, roomMgmt)
	if err != nil {
		log.Fatal("HTTP-Flv Server Start Faild:%v", err)
	}
	flvServer.Start()

	// TODO: hls server

	// Wait for interrupt signal to gracefully shutdown the server.
	quit := make(chan os.Signal)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	for {
		signal := <-quit
		switch signal {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			rtmpCloseFunc()
			flvCloseFunc()
			return
		case syscall.SIGHUP:
		default:
			return
		}
	}
}
