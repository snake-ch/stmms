package live

import (
	"strconv"
	"time"

	"gosm/pkg/log"
	"gosm/pkg/protocol/httpflv"
	"gosm/pkg/protocol/rtmp"
)

// OnRTMPPublish .
func (mgmt *RoomMgmt) OnRTMPPublish(stream *rtmp.NetStream) error {
	info, ok := stream.Info().(*rtmp.PublishInfo)
	if !ok {
		log.Error("Publisher: information TYPE error")
	}

	// release the old one if exist
	room, exist := mgmt.loadOrStore(info.Name)
	if exist {
		log.Debug("Publisher: live room '%s' exists, republish", info.Name)
		mgmt.OnRTMPUnPublish(stream)
	}
	var err error
	room.Publisher, err = NewPublisher(stream)
	if err != nil {
		return err
	}

	// start to serve
	go room.serve()
	return nil
}

// OnRTMPUnPublish .
func (mgmt *RoomMgmt) OnRTMPUnPublish(stream *rtmp.NetStream) error {
	info, ok := stream.Info().(*rtmp.PublishInfo)
	if !ok {
		log.Error("Publisher: information TYPE error")
	}

	room := mgmt.load(info.Name)
	if room != nil && room.Publisher != nil {
		log.Debug("Publisher: live room '%s' unpublish detected, release stream", info.Name)
		room.Publisher.close()
		room.Publisher = nil
	}
	return nil
}

// OnRTMPSubscribe .
func (mgmt *RoomMgmt) OnRTMPSubscribe(stream *rtmp.NetStream) error {
	info, ok := stream.Info().(*rtmp.SubscribeInfo)
	if !ok {
		log.Error("Subscriber: information TYPE error")
	}

	// check room if exist
	room, exist := mgmt.loadOrStore(info.StreamName)
	if !exist {
		log.Debug("Subscriber: live room '%s' not exist, waiting for av packet", info.StreamName)
	}

	// TODO: should check subscriber if exist ???

	// create subscriber
	uuid := mgmt.idworker.NextID()
	subscriber := &Subscriber{
		status: _statusNew,
		writer: stream,
		info: &SubscriberInfo{
			ID:            strconv.FormatInt(uuid, 10),
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
	return nil
}

// OnHTTPFlvSubscribe .
func (mgmt *RoomMgmt) OnHTTPFlvSubscribe(stream *httpflv.NetStream) error {
	// check room if exist
	room, exist := mgmt.loadOrStore(stream.Info().Stream)
	if !exist {
		log.Debug("Subscriber: live room '%s' not exist, waiting for av packet", stream.Info().Stream)
	}

	// TODO: should check subscriber if exist ???

	// create subscriber
	uuid := mgmt.idworker.NextID()
	subscriber := &Subscriber{
		status: _statusNew,
		writer: stream,
		info: &SubscriberInfo{
			ID:            strconv.FormatInt(uuid, 10),
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
	return nil
}

// OnHLSSubscribe .
func (mgmt *RoomMgmt) OnHLSSubscribe() error {
	return nil
}
