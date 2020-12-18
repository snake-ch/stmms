package live

import (
	"strconv"
	"time"

	"gosm/pkg/log"
	"gosm/pkg/protocol/httpflv"
	"gosm/pkg/protocol/rtmp"
	"gosm/pkg/utils"
)

// OnRTMPPublish .
func (mgmt *RoomMgmt) OnRTMPPublish(stream *rtmp.NetStream) error {
	info, _ := stream.Info().(*rtmp.PublishInfo)

	// release the old one if exist
	room, exist := mgmt.loadOrStore(info.Name)
	if exist {
		log.Debug("Publisher: live room '%s' exists, republish", info.Name)
		if err := mgmt.OnRTMPUnPublish(stream); err != nil {
			return err
		}
	}

	// publisher starts to serve
	var err error
	if room.Publisher, err = NewPublisher(stream); err != nil {
		return err
	}
	go room.serve()
	return nil
}

// OnRTMPUnPublish .
func (mgmt *RoomMgmt) OnRTMPUnPublish(stream *rtmp.NetStream) error {
	info, _ := stream.Info().(*rtmp.PublishInfo)
	room := mgmt.load(info.Name)
	if room != nil && room.Publisher != nil {
		log.Debug("Publisher: live room '%s' unpublish detected", info.Name)
		room.Publisher.close()
		room.Publisher = nil
	}
	return nil
}

// OnRTMPSubscribe .
func (mgmt *RoomMgmt) OnRTMPSubscribe(stream *rtmp.NetStream) error {
	info, _ := stream.Info().(*rtmp.SubscribeInfo)
	// check room if exist
	room, exist := mgmt.loadOrStore(info.StreamName)
	if !exist {
		log.Debug("Subscriber: live room '%s' not published yet, waiting for av packets", info.StreamName)
	}

	// TODO: should check subscriber if exist ???

	// create subscriber
	uuid := utils.Snowflake.NextID()
	subscriber := &Subscriber{
		status: _new,
		writer: stream,
		info: &SubscriberInfo{
			UID:           strconv.FormatInt(uuid, 10),
			Protocol:      _Rtmp,
			Type:          _TypeLive,
			SubscribeTime: time.Now(),
		},
	}

	// fetch av metadata if publisher exist
	if room.Publisher != nil {
		metadataPacket, err := room.Publisher.metadata()
		if err != nil {
			return err
		}
		if err := subscriber.writer.WriteAVPacket(metadataPacket); err != nil {
			return err
		}
	}
	room.Subscribers.Store(uuid, subscriber)
	return nil
}

// OnRTMPUnSubsribe .
func (mgmt *RoomMgmt) OnRTMPUnSubsribe(stream *rtmp.NetStream) error {
	return stream.Close()
}

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
		status: _new,
		writer: stream,
		info: &SubscriberInfo{
			UID:           strconv.FormatInt(uuid, 10),
			Protocol:      _HTTPFlv,
			Type:          _TypeLive,
			SubscribeTime: time.Now(),
		},
	}

	// fetch av metadata if publisher exist
	if room.Publisher != nil {
		metadataPacket, err := room.Publisher.metadata()
		if err != nil {
			return err
		}
		if err := subscriber.writer.WriteAVPacket(metadataPacket); err != nil {
			return err
		}
	}
	room.Subscribers.Store(uuid, subscriber)
	return nil
}

// OnHTTPFlvUnSubscribe .
func (mgmt *RoomMgmt) OnHTTPFlvUnSubscribe(stream *httpflv.NetStream) error {
	return stream.Close()
}

// OnHLSSubscribe .
func (mgmt *RoomMgmt) OnHLSSubscribe() error {
	return nil
}
