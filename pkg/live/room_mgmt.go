package live

import (
	"sync"

	"gosm/pkg/avformat"
	"gosm/pkg/log"
)

// RoomMgmt living room managerment, defined followed:
//    room name <=> publish stream name
//		-> map[room's name]*room
//		-> map[publisher's name]map[subscriber's name]*subscriber
type RoomMgmt struct {
	rooms *sync.Map
}

// NewRoomMgmt .
func NewRoomMgmt() (*RoomMgmt, error) {
	return &RoomMgmt{rooms: &sync.Map{}}, nil
}

// find room and get
func (mgmt *RoomMgmt) load(name string) *Room {
	if room, exist := mgmt.rooms.Load(name); exist {
		return room.(*Room)
	}
	return nil
}

// find room and get, create room if not exist
func (mgmt *RoomMgmt) loadOrStore(name string) (*Room, bool) {
	room, exist := mgmt.rooms.LoadOrStore(name, &Room{
		Publisher:   nil,
		Subscribers: &sync.Map{},
	})
	return room.(*Room), exist
}

// RoomInfo .
type RoomInfo struct {
	Name            string
	Type            string
	PublisherInfo   *PublisherInfo
	SubscribersInfo []*SubscriberInfo
}

// Room living room
type Room struct {
	Publisher   *Publisher //
	Subscribers *sync.Map  // <=> map[subscriber's name]*subscriber
}

// find subscriber
func (room *Room) loadSubscriber(name string) (*Subscriber, bool) {
	if subscriber, exist := room.Subscribers.Load(name); exist {
		return subscriber.(*Subscriber), true
	}
	return nil, false
}

// room start publisher, loop to broadcast av packets
func (room *Room) serve() {
	publisher := room.Publisher
	defer func() {
		log.Info("Room: app '%s', stream '%s' stop publishing", publisher.info.AppName, publisher.info.StreamName)
	}()
	log.Info("Room: app '%s' stream '%s' start publishing", publisher.info.AppName, publisher.info.StreamName)

	for {
		select {
		case <-publisher.ctx.Done():
			return
		default:
			packet, err := publisher.reader.ReadAVPacket()
			if err != nil {
				return
			}
			switch packet.TypeID {
			case avformat.TypeMetadataAMF0: // metadata
				if err := publisher.parseMetadata(packet); err != nil {
					log.Error("Publisher: parses metadata error, %v", err)
					return
				}
				metaPacket, _ := publisher.metadata()
				room.Subscribers.Range(room.broadcast(metaPacket))
			case avformat.TypeAudio: // audio
				fallthrough
			case avformat.TypeVideo: // video
				publisher.cache.Write(packet)
				room.Subscribers.Range(room.broadcast(packet))
			}
		}
	}
}

// broadcast av packet to all subscribers
func (room *Room) broadcast(packet *avformat.AVPacket) func(key, value interface{}) bool {
	return func(key, value interface{}) bool {
		subscriber := value.(*Subscriber)

		var err error
		switch subscriber.status {
		case _new: // flush publisher's cache av packets
			err = room.Publisher.cache.WriteTo(subscriber.writer)
			subscriber.status = _running
		case _running: // flush av packet
			err = subscriber.writer.WriteAVPacket(packet)
		case _closed:
			room.Subscribers.Delete(key)
		}

		if err != nil {
			log.Error("Room: subscriber '%s' writes av packet error, %v, remove it", subscriber.info.UID, err)
			subscriber.close()
			room.Subscribers.Delete(key)
		}
		return true
	}
}
