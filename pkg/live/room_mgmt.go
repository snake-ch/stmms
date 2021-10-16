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
func NewRoomMgmt() *RoomMgmt {
	return &RoomMgmt{rooms: &sync.Map{}}
}

// find room and get
func (mgmt *RoomMgmt) load(name string) *Room {
	if room, exist := mgmt.rooms.Load(name); exist {
		return room.(*Room)
	}
	return nil
}

// find room and get, create new if not exist
func (mgmt *RoomMgmt) loadOrStore(name string) (*Room, bool) {
	room, exist := mgmt.rooms.LoadOrStore(name, &Room{
		Publisher:          nil, // lazy created
		RTMPSubscribers:    &sync.Map{},
		HTTPFlvSubscribers: &sync.Map{},
		HLSSubscriber:      nil, // lazy created
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
	Publisher          *Publisher
	RTMPSubscribers    *sync.Map   // <=> map[subscriber's name]*subscriber
	HTTPFlvSubscribers *sync.Map   // <=> map[subscriber's name]*subscriber
	HLSSubscriber      *Subscriber // hls subscriber
}

// find subscriber
func (room *Room) loadSubscriber(name string) (*Subscriber, bool) {
	if subscriber, exist := room.RTMPSubscribers.Load(name); exist {
		return subscriber.(*Subscriber), true
	}
	if subscriber, exist := room.HTTPFlvSubscribers.Load(name); exist {
		return subscriber.(*Subscriber), true
	}
	return nil, false
}

// room start to publish, loop to broadcast av packets
func (room *Room) serve() {
	publisher := room.Publisher
	defer func() {
		log.Info("Room: app '%s', stream '%s' stop publishing", publisher.info.AppName, publisher.info.StreamName)
		room.Close()
	}()
	log.Info("Room: app '%s', stream '%s' start publishing", publisher.info.AppName, publisher.info.StreamName)

	for {
		packet, err := publisher.rc.ReadAVPacket()
		if err != nil {
			return
		}

		// HLS
		if room.HLSSubscriber != nil {
			if err := room.HLSSubscriber.wc.WriteAVPacket(packet); err != nil {
				room.HLSSubscriber.Close()
			}
		}

		// RTMP & HTTL-FLV
		switch packet.TypeID {
		case avformat.TypeMetadataAMF0: // metadata
			if err := publisher.parseMetadata(packet); err != nil {
				log.Error("Publisher: parse metadata error, %v", err)
				return
			}
			metaPacket, _ := publisher.metadata()
			room.RTMPSubscribers.Range(room.broadcast(room.RTMPSubscribers, metaPacket))
			room.HTTPFlvSubscribers.Range(room.broadcast(room.HTTPFlvSubscribers, metaPacket))
		case avformat.TypeAudio: // audio
			fallthrough
		case avformat.TypeVideo: // video
			publisher.cache.Write(packet)
			room.RTMPSubscribers.Range(room.broadcast(room.RTMPSubscribers, packet))
			room.HTTPFlvSubscribers.Range(room.broadcast(room.HTTPFlvSubscribers, packet))
		}
	}
}

// broadcast av packet to all subscribers
func (room *Room) broadcast(m *sync.Map, packet *avformat.AVPacket) func(key, value interface{}) bool {
	return func(key, value interface{}) bool {
		subscriber := value.(*Subscriber)

		var err error
		switch subscriber.status {
		case New: // flush gop cache
			err = room.Publisher.cache.WriteTo(subscriber.wc)
			subscriber.status = Running
		case Running: // flush av packet
			err = subscriber.wc.WriteAVPacket(packet)
		case Closed:
			m.Delete(key)
		}

		if err != nil {
			log.Error("Room: subscriber '%s' writes av packet error, %v, remove it", subscriber.info.UID, err)
			subscriber.Close()
			m.Delete(key)
		}
		return true
	}
}

// Close
func (room *Room) Close() error {
	// close publisher
	if room.Publisher != nil {
		room.Publisher.Close()
	}

	// close rtmp subscribers
	room.RTMPSubscribers.Range(func(key, value interface{}) bool {
		value.(*Subscriber).Close()
		return true
	})

	// close http-flv subscribers
	room.HTTPFlvSubscribers.Range(func(key, value interface{}) bool {
		value.(*Subscriber).Close()
		return true
	})

	// close hls subscriber
	if room.HLSSubscriber != nil {
		room.HLSSubscriber.Close()
	}

	return nil
}
