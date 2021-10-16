package live

import (
	"fmt"

	"gosm/pkg/avformat"
)

const GopMax = 128

/************************************/
/********** AV Packet Cache *********/
/************************************/

// AVCache .
type AVCache struct {
	audioConfig *avformat.AVPacket // audio parameter sets
	videoConfig *avformat.AVPacket // video parameter sets
	gopGroup    *GopGroup
}

// NewAVCache .
func NewAVCache(gopSize uint8) *AVCache {
	return &AVCache{
		audioConfig: nil,
		videoConfig: nil,
		gopGroup:    NewGopGroup(gopSize),
	}
}

// Write
func (cache *AVCache) Write(packet *avformat.AVPacket) error {
	if packet.IsAACSeqHeader() {
		cache.audioConfig = packet
		return nil
	}
	if packet.IsAVCSeqHeader() || packet.IsHEVCSeqHeader() {
		cache.videoConfig = packet
		return nil
	}
	return cache.gopGroup.Write(packet)
}

// WriteTo flush data to subscriber
func (cache *AVCache) WriteTo(wc AVWriteCloser) error {
	// audio config
	if cache.audioConfig != nil {
		if err := wc.WriteAVPacket(cache.audioConfig); err != nil {
			return err
		}
	}
	// video config
	if cache.videoConfig != nil {
		if err := wc.WriteAVPacket(cache.videoConfig); err != nil {
			return err
		}
	}
	// gop group
	if err := cache.gopGroup.WriteTo(wc); err != nil {
		return err
	}
	return nil
}

/************************************/
/**************** GOP ***************/
/************************************/

// GOP .
type gop struct {
	packets []*avformat.AVPacket
}

// NewGOP .
func newGop() *gop {
	gop := &gop{
		packets: make([]*avformat.AVPacket, 0),
	}
	return gop
}

// reset gop
func (gop *gop) reset() {
	gop.packets = gop.packets[:0]
}

// cache av packet
func (gop *gop) write(packet *avformat.AVPacket) error {
	if len(gop.packets) >= GopMax {
		return fmt.Errorf("GOP: large than maxinum capacity %d", GopMax)
	}
	gop.packets = append(gop.packets, packet)
	return nil
}

// write gop cache av packets to subscriber
func (gop *gop) writeTo(wc AVWriteCloser) error {
	for _, packet := range gop.packets {
		if err := wc.WriteAVPacket(packet); err != nil {
			return err
		}
	}
	return nil
}

// GopCache group of GOP
type GopGroup struct {
	capacity uint8
	current  uint8
	gops     []*gop
}

func NewGopGroup(capacity uint8) *GopGroup {
	group := &GopGroup{
		capacity: capacity,
		current:  0,
		gops:     make([]*gop, capacity),
	}
	return group
}

// Write cache av packet
func (group *GopGroup) Write(packet *avformat.AVPacket) error {
	if group.capacity == 0 {
		return nil
	}

	if packet.IsAVCSeqHeader() || packet.IsHEVCSeqHeader() {
		return nil
	}

	// IDR frame, use next gop create if not exist, else reset
	if packet.IsAVCKeyframe() || packet.IsHEVCKeyframe() {
		group.current = (group.current + 1) % group.capacity
		if gop := group.gops[group.current]; gop != nil {
			gop.reset()
		} else {
			group.gops[group.current] = newGop()
		}
	}

	// cache IDR or B or P frame
	gop := group.gops[group.current]
	if gop != nil {
		gop.write(packet)
	}
	return nil
}

// WriteTo write gops cache av packets to subscriber
func (group *GopGroup) WriteTo(wc AVWriteCloser) error {
	for idx := uint8(0); idx < group.capacity; idx++ {
		pos := (group.current + 1 + idx) % group.capacity
		if gop := group.gops[pos]; gop != nil {
			if err := gop.writeTo(wc); err != nil {
				return err
			}
		}
	}
	return nil
}
