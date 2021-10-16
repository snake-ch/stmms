package udp

import (
	"fmt"
	"gosm/pkg/avformat"
	"gosm/pkg/log"
	"gosm/pkg/protocol/rtsp/rtcp"
	"gosm/pkg/protocol/rtsp/rtp"
	"net"
	"strconv"
	"time"
)

type Session struct {
	rtp  *rtp.Connection
	rtcp *rtcp.Connection

	ssrc      uint32
	sysTs     int64  // local timestamp
	rtpBaseTs uint32 // first rtp packet's timestamp
	rtpLastTs uint32 // last rtp packet's timestamp

	depacketizer  *rtp.Depacketizer
	packer        *rtp.RTMPPacker
	avcSeqHdrSent bool
	aacSeqHdrSent bool
	avQueue       chan *avformat.AVPacket
}

func NewSession(port string) (*Session, error) {
	session := &Session{
		rtcp:          nil,
		rtp:           nil,
		ssrc:          0,
		sysTs:         time.Now().UnixNano() / 1e6,
		rtpBaseTs:     0,
		rtpLastTs:     0,
		depacketizer:  rtp.NewDepacketizer(),
		packer:        rtp.NewRTMPRepacker(),
		avcSeqHdrSent: false,
		aacSeqHdrSent: false,
		avQueue:       make(chan *avformat.AVPacket, 1024),
	}

	// rtp
	rtpPort, err := strconv.Atoi(port)
	if err != nil {
		return nil, err
	}
	if rtpPort%2 == 1 {
		return nil, fmt.Errorf("UDP: only accept even number as rtp port")
	}
	rtpAddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(rtpPort))
	if err != nil {
		return nil, fmt.Errorf("UDP: address resolve error, %v", err)
	}
	rtpConn, err := net.ListenUDP("udp", rtpAddr)
	if err != nil {
		return nil, fmt.Errorf("UDP: listen rtp port error, %v", err)
	}
	session.rtp = rtp.NewConn(rtpConn)

	// rtcp
	rtcpPort, err := strconv.Atoi(port)
	if err != nil {
		return nil, err
	}
	rtcpAddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(rtcpPort+1))
	if err != nil {
		return nil, fmt.Errorf("UDP: address resolve error, %v", err)
	}
	rtcpConn, err := net.ListenUDP("udp", rtcpAddr)
	if err != nil {
		return nil, fmt.Errorf("UDP: listen rtp port error, %v", err)
	}
	session.rtcp = rtcp.NewConn(rtcpConn)

	log.Info("UDP: session listen rtp on %s, rtcp on %s", rtpConn.LocalAddr(), rtcpConn.LocalAddr())
	return session, nil
}

// Serve
func (session *Session) Serve() {
	go session.rtcp.Serve()
	go session.rtp.Serve()

	for {
		select {
		case rtcpPacket := <-session.rtcp.Queue():
			switch rtcpPacket.PT {
			case rtcp.SR:
				// sr := rtcpPacket.ParseSR()
			case rtcp.RR:
			case rtcp.SDES:
			case rtcp.BYE:
			case rtcp.APP:
			default:
				fmt.Printf("%+v\n", rtcpPacket)
				log.Error("RTCP: not support packet type: %d", rtcpPacket.PT)
			}

		case video := <-session.rtp.VideoQueue():
			if session.rtpBaseTs == 0 {
				session.rtpBaseTs = video.Timestamp()
			}
			session.rtpLastTs = video.Timestamp()

			marker, err := session.depacketizer.DepacketizeVideo(video)
			if err != nil {
				log.Error("RTP: depacketize video error, %v", err)
			}

			if marker {
				// video sequence header tag should come first
				if !session.avcSeqHdrSent {
					sps := session.depacketizer.SPS()
					pps := session.depacketizer.PPS()
					packet, err := session.packer.VideoSeqHdrPacket(sps, pps)
					if err != nil {
						log.Error("RTP: repack video sequence packet error, %v", err)
					}
					if packet != nil {
						session.avQueue <- packet
						session.avcSeqHdrSent = true
					}
				}

				// video full frame
				// NOTE: due to h264 with B frame, rtp timestamp increases not monotonic,
				// 			 not support to sort video frame for now, ensure pulling without B frame
				// TODO: fix it
				payload := session.depacketizer.Nalus()
				ts := uint32(time.Now().UnixNano()/1e6 - session.sysTs)
				pts := int32(session.rtpLastTs - session.rtpBaseTs)
				dts := pts
				avPacket, err := session.packer.PackVideo(ts, pts, dts, payload)
				if err != nil {
					log.Error("RTP: repack video nalu packet error, %v", err)
				}
				session.avQueue <- avPacket
			}
		case audio := <-session.rtp.AudioQueue():
			if session.rtpBaseTs == 0 {
				session.rtpBaseTs = audio.Timestamp()
			}
			session.rtpLastTs = audio.Timestamp()

			// audio sequence header tag should come first
			if !session.aacSeqHdrSent {
				session.avQueue <- session.packer.AudioSeqHdrPacket()
				session.aacSeqHdrSent = true
			}

			// NOTE: few format supported
			audioFrames, err := session.depacketizer.DepacketizeAudio(audio)
			if err != nil {
				log.Error("RTP: depacketize audio error, %v", err)
			}
			for _, frame := range audioFrames {
				session.avQueue <- session.packer.PackAudio(frame.Timestamp, frame.Raw)
			}
		}
	}
}

/************************************/
/********* Publish Interface ********/
/************************************/

// ReadAVPacket .
func (session *Session) ReadAVPacket() (*avformat.AVPacket, error) {
	avPacket, ok := <-session.avQueue
	if !ok {
		return nil, fmt.Errorf("RTP: stream '%d' media buffer closed", session.ssrc)
	}
	return avPacket, nil
}

// Close .
func (session *Session) Close() error {
	if err := session.rtp.Close(); err != nil {
		return err
	}
	if err := session.rtcp.Close(); err != nil {
		return err
	}
	return nil
}
