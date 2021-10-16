package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gosm/pkg/config"
	"gosm/pkg/live"
	"gosm/pkg/log"
	"gosm/pkg/protocol/hls"
	"gosm/pkg/protocol/httpflv"
	"gosm/pkg/protocol/rtmp"
	"gosm/pkg/protocol/rtsp/udp"
)

func main() {
	fmt.Printf(`
      _____       _____ __  __ 
     / ____|     / ____|  \/  |
    | |  __  ___| (___ | \  / |
    | | |_ |/ _ \\___ \| |\/| |
    | |__| | (_) |___) | |  | |
     \_____|\___/_____/|_|  |_|    version: %s`, config.Version+"\n\n")

	// log level
	log.SetLevel(config.Global.LogLevel)

	// living room managerment
	roomMgmt := live.NewRoomMgmt()

	// rtmp server
	rtmpServer, rtmpCloseFunc, err := rtmp.NewServer("tcp", ":"+config.Global.RTMP.Port)
	if err != nil {
		log.Fatal("RTMP Server Starts Faild:%v", err)
	}
	rtmpServer.SetObserver(roomMgmt)
	rtmpServer.Serve()

	// http-flv server
	flvServer, flvCloseFunc, err := httpflv.NewServer("tcp", ":"+config.Global.HTTPFLV.Port)
	if err != nil {
		log.Fatal("HTTP-Flv Server Starts Faild:%v", err)
	}
	flvServer.SetObserver(roomMgmt)
	flvServer.Serve()

	// hls server
	hlsServer, hlsCloseFunc, err := hls.NewServer("tcp", ":"+config.Global.HLS.Port)
	if err != nil {
		log.Fatal("HLS Server Starts Faild:%v", err)
	}
	hlsServer.Serve()

	// udp server
	session, err := udp.NewSession("50000")
	if err != nil {
		log.Fatal("UDP Server Starts Faild:%v", err)
	}
	go session.Serve()

	// udp to rtmp client
	rtmpClient, err := rtmp.NewClient(config.Global.RTP.Remote)
	if err != nil {
		log.Fatal("RTMP Client Initials Faild:%v", err)
	}
	if err := rtmpClient.Handshake(); err != nil {
		log.Fatal("%+v", err)
	}
	if err := rtmpClient.Connect(); err != nil {
		log.Fatal("%+v", err)
	}
	if err := rtmpClient.Publish(); err != nil {
		log.Fatal("%+v", err)
	}

	go func() {
		for {
			avPacket, err := session.ReadAVPacket()
			if err != nil {
				return
			}
			if err := rtmpClient.WriteAVPacket(avPacket); err != nil {
				return
			}
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server.
	quit := make(chan os.Signal, 1)
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
			hlsCloseFunc()
			return
		case syscall.SIGHUP:
		default:
			return
		}
	}
}
