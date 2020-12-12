package live

import (
	"strconv"
	"time"

	"gosm/pkg/log"
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
	if room == nil {
		log.Debug("Publisher: live room '%s' not exist, ignore release stream", info.Name)
	} else {
		log.Debug("Publisher: live room '%s' unpublish detected, release stream", info.Name)
		room.Publisher.cancel()
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
		stream: stream,
		info: &SubscriberInfo{
			ID:            strconv.FormatInt(uuid, 10),
			Protocol:      _Rtmp,
			Type:          _TypeLive,
			SubscribeTime: time.Now(),
		},
	}

	// fetch av metadata if publisher exist
	if room.Publisher != nil {
		metadataPacket, _ := room.Publisher.metadata()
		subscriber.stream.WriteAVPacket(metadataPacket)
	}
	room.Subscribers.Store(uuid, subscriber)
	return nil
}

// OnRTMPUnSubsribe .
func (mgmt *RoomMgmt) OnRTMPUnSubsribe(stream *rtmp.NetStream) error {
	return nil
}
