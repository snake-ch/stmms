package rtmp

// ChunkStream .
type ChunkStream struct {
	id           uint32       // chunk stream id
	preHeader    *ChunkHeader // previous header
	cacheMessage *Message     // receiving message
}
