package live

import (
	"sync"

	"gosm/pkg/avformat"
	"gosm/pkg/avformat/flv"
	"gosm/pkg/log"
	"gosm/pkg/utils"
)

// RoomMgmt living room managerment, defined followed:
//    room name <=> publish stream name
//		-> map[room's name]*room
//		-> map[publisher's name]map[subscriber's name]*subscriber
type RoomMgmt struct {
	idworker *utils.IDWorker
	rooms    sync.Map
}

// NewRoomMgmt .
func NewRoomMgmt(idworker *utils.IDWorker) (*RoomMgmt, error) {
	return &RoomMgmt{idworker: idworker}, nil
}

// find room and get
func (mgmt *RoomMgmt) load(name string) *Room {
	if room, exist := mgmt.rooms.Load(name); !exist {
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
		log.Debug("Room: app '%s', stream '%s' stops", publisher.info.AppName, publisher.info.StreamName)
	}()
	log.Debug("Room: App '%s' Stream '%s' starts", publisher.info.AppName, publisher.info.StreamName)

	for {
		select {
		case <-publisher.ctx.Done():
			return
		default:
			packet, _ := publisher.reader.ReadAVPacket()
			switch packet.TypeID {
			case avformat.TypeMetadata: // metadata
				if err := publisher.parseMetadata(packet); err != nil {
					log.Error("Publisher: parses metadata error, %v", err)
					break
				}
				metaPacket, _ := publisher.metadata()
				room.Subscribers.Range(room.broadcast(metaPacket))
			case avformat.TypeAudio: // audio
				if packet.IsAACSeqHeader() {
					// TODO: parse aac header
				}
				if packet.IsAACRaw() {
					// TODO: handle aac raw
				}
				room.Subscribers.Range(room.broadcast(packet))
			case avformat.TypeVideo: // video
				if packet.IsAVCSeqHeader() || packet.IsHEVCSeqHeader() {
					log.Info("AVCSeqHeader || HEVCSeqHeader")
					seqHeader, err := flv.ParseVideoSeqHeader(packet.Body)
					if err != nil {
						break
					}
					copy(publisher.sps, seqHeader.Sps)
					copy(publisher.pps, seqHeader.Pps)
				}
				if packet.IsAVCKeyframe() || packet.IsHEVCKeyframe() {
					// TODO: cache video gop
				}
				room.Subscribers.Range(room.broadcast(packet))
			}
		}
	}
}

// broadcast av packet to all subscribers
func (room *Room) broadcast(packet *avformat.AVPacket) func(key, value interface{}) bool {
	return func(key, value interface{}) bool {
		subscriber := value.(*Subscriber)
		stream := subscriber.stream.(AVWriteCloser)
		if err := stream.WriteAVPacket(packet); err != nil {
			log.Error("Subscribe: write av packet err, %v, close it", err)
			stream.Close()
			room.Subscribers.Delete(key)
		}
		return true
	}
}
