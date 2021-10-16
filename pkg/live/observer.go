package live

import (
	"fmt"
	"strconv"
	"time"

	"gosm/pkg/config"
	"gosm/pkg/log"
	"gosm/pkg/protocol/hls"
	"gosm/pkg/protocol/httpflv"
	"gosm/pkg/protocol/rtmp"
	"gosm/pkg/utils"
)

/***********************************
 ********** RTMP Observer **********
 ***********************************/

// OnRTMPPublish .
func (mgmt *RoomMgmt) OnRTMPPublish(stream *rtmp.NetStream) error {
	// release the old one if exist
	info := stream.Info()
	room, exist := mgmt.loadOrStore(info.Name)
	if exist {
		log.Debug("Publisher: live room '%s' exists, try to republish", info.Name)
		if err := mgmt.OnRTMPUnPublish(stream); err != nil {
			return err
		}
	}

	// publish rtmp
	room.Publisher = &Publisher{
		info: &PublisherInfo{
			AppName:     stream.ConnInfo().App,
			StreamName:  info.Name,
			StreamType:  info.Type,
			PublishTime: time.Now(),
			MetaData:    nil,
		},
		cache: NewAVCache(config.Global.RTMP.GopSize),
		rc:    stream,
	}

	// publish hls
	if config.Global.HLS.Enable {
		hlsStrem, err := hls.NewNetStream(stream.ConnInfo().App, info.Name)
		if err != nil {
			return err
		}
		if err = mgmt.OnHLSSubscribe(hlsStrem); err != nil {
			return err
		}
	}

	go room.serve()
	return nil
}

// OnRTMPUnPublish .
func (mgmt *RoomMgmt) OnRTMPUnPublish(stream *rtmp.NetStream) error {
	info := stream.Info()
	room := mgmt.load(info.Name)
	if room != nil && room.Publisher != nil {
		log.Debug("Publisher: live room '%s' unpublish", info.Name)
		room.Publisher.Close()
		room.Publisher = nil
	}
	return nil
}

// OnRTMPSubscribe .
func (mgmt *RoomMgmt) OnRTMPSubscribe(stream *rtmp.NetStream) error {
	info := stream.Info()
	// check room if exist
	room, exist := mgmt.loadOrStore(info.StreamName)
	if !exist {
		log.Debug("Subscriber: live room '%s' not exist, creating...", info.StreamName)
	}

	// TODO: should check subscriber if exist ???

	// create subscriber
	uuid := utils.Snowflake.NextID()
	subscriber := &Subscriber{
		status: New,
		wc:     stream,
		info: &SubscriberInfo{
			UID:           strconv.FormatInt(uuid, 10),
			Protocol:      RTMP,
			Type:          TypeLive,
			SubscribeTime: time.Now(),
		},
	}

	// fetch av metadata if publisher exist
	if room.Publisher != nil {
		metadataPacket, err := room.Publisher.metadata()
		if err != nil {
			return err
		}
		if err := subscriber.wc.WriteAVPacket(metadataPacket); err != nil {
			return err
		}
	}
	room.RTMPSubscribers.Store(uuid, subscriber)
	return nil
}

// OnRTMPUnSubsribe .
func (mgmt *RoomMgmt) OnRTMPUnSubsribe(stream *rtmp.NetStream) error {
	return stream.Close()
}

/***********************************
 ******** HTTP-FLV Observer ********
 ***********************************/

// OnHTTPFlvSubscribe .
func (mgmt *RoomMgmt) OnHTTPFlvSubscribe(stream *httpflv.NetStream) error {
	// check room if exist
	room, exist := mgmt.loadOrStore(stream.Info().Stream)
	if !exist {
		log.Debug("Subscriber: live room '%s' not published yet, waiting for av packets", stream.Info().Stream)
	}

	// TODO: should check subscriber if exist ???

	// create subscriber
	uuid := utils.Snowflake.NextID()
	subscriber := &Subscriber{
		status: New,
		wc:     stream,
		info: &SubscriberInfo{
			UID:           strconv.FormatInt(uuid, 10),
			Protocol:      HTTPFLV,
			Type:          TypeLive,
			SubscribeTime: time.Now(),
		},
	}

	// fetch av metadata if publisher exist
	if room.Publisher != nil {
		metadata, err := room.Publisher.metadata()
		if err != nil {
			return err
		}
		if err := subscriber.wc.WriteAVPacket(metadata); err != nil {
			return err
		}
	}
	room.HTTPFlvSubscribers.Store(uuid, subscriber)
	return nil
}

// OnHTTPFlvUnSubscribe .
func (mgmt *RoomMgmt) OnHTTPFlvUnSubscribe(stream *httpflv.NetStream) error {
	return stream.Close()
}

/***********************************
 *********** HLS Observer **********
 ***********************************/

// OnHLSSubscribe .
func (mgmt *RoomMgmt) OnHLSSubscribe(stream *hls.NetStream) error {
	// check room if exist
	room, exist := mgmt.loadOrStore(stream.Info().Stream)
	if !exist {
		return fmt.Errorf("Subscriber: live room '%s' not published yet, ingore HLS", stream.Info().Stream)
	}

	// create subscriber
	uuid := utils.Snowflake.NextID()
	subscriber := &Subscriber{
		status: Running,
		wc:     stream,
		info: &SubscriberInfo{
			UID:           strconv.FormatInt(uuid, 10),
			Protocol:      HLS,
			Type:          TypeLive,
			SubscribeTime: time.Now(),
		},
	}
	room.HLSSubscriber = subscriber

	return nil
}
