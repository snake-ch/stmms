package live

import (
	"fmt"
	"gosm/pkg/avformat"
)

const _MaxSize = 512

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
	if len(gop.packets) >= _MaxSize {
		return fmt.Errorf("GOP: large than maxinum capacity %d", _MaxSize)
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
type GopCache struct {
	capacity uint8
	current  uint8
	gops     []*gop
}

// NewGopCache capacity should >= 2, ping-pong operation.
func NewGopCache(capacity uint8) *GopCache {
	group := &GopCache{
		capacity: capacity,
		current:  0,
		gops:     make([]*gop, capacity),
	}
	return group
}

// Write cache av packet
func (group *GopCache) Write(packet *avformat.AVPacket) error {
	if !packet.IsAVC() || !packet.IsHEVC() || packet.IsAVCSeqHeader() || packet.IsHEVCSeqHeader() {
		return nil
	}

	// IDR frame, use next gop, create if not exist, reset if exist
	if packet.IsAVCKeyframe() || packet.IsHEVCKeyframe() {
		group.current = (group.current + 1) % group.capacity
		if gop := group.gops[group.current]; gop == nil {
			group.gops[group.current] = newGop()
		} else {
			gop.reset()
		}
	}

	// cache IDR or B or P frame
	if gop := group.gops[group.current]; gop != nil {
		gop.write(packet)
	}
	return nil
}

// WriteTo write gops cache av packets to subscriber
func (group *GopCache) WriteTo(wc AVWriteCloser) error {
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
