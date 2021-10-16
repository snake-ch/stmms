package hls

import (
	"fmt"
	"gosm/pkg/config"
	"gosm/pkg/utils"
	"os"
	"strconv"
)

var tsPath = config.Global.HLS.TsPath
var tsPrefix = config.Global.HLS.TsPrefix
var duration = config.Global.HLS.TsDuration
var winSize = config.Global.HLS.TsWindow / config.Global.HLS.TsDuration

type TSSegment struct {
	ID       int
	Duration float64
}

type M3U8 struct {
	path          string       // ts files path
	prefix        string       // ts file prefix
	stream        string       // stream name
	lastTimestamp uint32       // last ts segment(I-frame) timestamp
	sn            int          // current ts segment serial number
	sequence      int          // m3u8 field EXT-X-MEDIA-SEQUENCE
	segments      []*TSSegment // ts segments
}

func NewM3U8(stream string) *M3U8 {
	m3u8 := &M3U8{
		path:          tsPath,
		prefix:        tsPrefix,
		stream:        stream,
		lastTimestamp: 0,
		sn:            0,
		sequence:      0,
		segments:      make([]*TSSegment, 0),
	}
	return m3u8
}

func (m3u8 *M3U8) NextSegment() string {
	return m3u8.prefix + m3u8.stream + "-" + strconv.Itoa(m3u8.sn) + ".ts"
}

// Check check should cut ts segment
func (m3u8 *M3U8) Check(timestamp uint32) bool {
	return timestamp-m3u8.lastTimestamp > uint32(duration) // not strictly
}

// Update cache segment info, generate playlist, ready for next segment
func (m3u8 *M3U8) Update(timestamp uint32) (bool, error) {
	if ok := m3u8.Check(timestamp); !ok {
		return false, nil
	}

	// cache segment information
	segment := &TSSegment{
		ID:       m3u8.sn,
		Duration: float64(timestamp-m3u8.lastTimestamp) / 1000,
	}
	if len(m3u8.segments) < winSize {
		m3u8.segments = append(m3u8.segments, segment)
	} else {
		m3u8.segments = append(m3u8.segments[1:], segment)
	}

	// generate play list
	if err := m3u8.GenMediaPlaylist(); err != nil {
		return false, err
	}

	// next segment
	m3u8.sn += 1
	if m3u8.sn > winSize {
		m3u8.sequence = m3u8.sn - winSize
	}
	m3u8.lastTimestamp = timestamp
	return true, nil
}

// MaxDuration .
func (m3u8 *M3U8) MaxDuration() (duration float64) {
	for _, segment := range m3u8.segments {
		if duration < segment.Duration { // not strictly
			duration = segment.Duration
		}
	}
	return
}

// GenMasterPlaylist .
func (m3u8 *M3U8) GenMasterPlaylist() error {
	return nil
}

// GenMediaPlaylist .
func (m3u8 *M3U8) GenMediaPlaylist() error {
	// temporary .m3u8
	fn := strconv.FormatInt(utils.Snowflake.NextID(), 10) + ".m3u8"
	fp, err := os.Create(fn)
	if err != nil {
		return err
	}

	// playlist base tag
	fp.WriteString("#EXTM3U\n")
	fp.WriteString("#EXT-X-VERSION:3\n")
	fp.WriteString("#EXT-X-ALLOW-CACHE:NO\n")
	fp.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%.3f\n", m3u8.MaxDuration()))
	fp.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n\n", m3u8.sequence))

	// media segment tags
	for _, segment := range m3u8.segments {
		duration := segment.Duration
		fn := m3u8.prefix + m3u8.stream + "-" + strconv.Itoa(segment.ID)
		if _, err := fp.WriteString(fmt.Sprintf("#EXTINF:%.3f,\n%s\n", duration, fn)); err != nil {
			return err
		}
	}

	// replace m3u8
	if err := fp.Close(); err != nil {
		return err
	}
	return os.Rename(fn, m3u8.stream+".m3u8")
}
