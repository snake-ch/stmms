package rtmp

// Message Type
const (
	SetChunkSize         = uint8(1)  // Set Chunk Size
	AbortMessage         = uint8(2)  // Abort Message
	Acknowledgement      = uint8(3)  // Acknowledgement
	UserControlMessages  = uint8(4)  // User Control Message
	WindowAckSize        = uint8(5)  // Window Acknowledgement Size
	SetPeerBandwidth     = uint8(6)  // Set Peer Bandwidth
	AudioType            = uint8(8)  // Audio Message
	VideoType            = uint8(9)  // Video Message
	AggregateMessageType = uint8(22) // Aggregate Message
	SharedObjectAmf0     = uint8(19) // Shared Object Message
	SharedObjectAmf3     = uint8(16) // Shared Object Message
	DataAmf0             = uint8(18) // Data Message
	DataAmf3             = uint8(15) // Data Message
	CommandAmf0          = uint8(20) // Command Message
	CommandAmf3          = uint8(17) // Command Message
)

// Stream ID
const (
	SIDNetConnnection = uint32(0)
	SIDNetStream      = uint32(1)
)

// Chunk Stream ID
const (
	CSIDProtocolControl = uint32(2)
	CSIDCommand         = uint32(3)
	CSIDUserControl     = uint32(4)
	CSIDMetadata        = uint32(5)
	CSIDAudio           = uint32(6)
	CSIDVideo           = uint32(7)
)

// User Control Message Events
const (
	EventStreamBegin      = uint16(0)
	EventStreamEOF        = uint16(1)
	EventStreamDry        = uint16(2)
	EventSetBufferLength  = uint16(3)
	EventStreamIsRecorded = uint16(4)
	EventPingRequest      = uint16(6)
	EventPingResponse     = uint16(7)
	EventRequestVerify    = uint16(0x1a)
	EventRespondVerify    = uint16(0x1b)
	EventBufferEmpty      = uint16(0x1f)
	EventBufferReady      = uint16(0x20)
)

// Bandwidth limit type
const (
	BindwidthLimitHard    = uint8(0)
	BindwidthLimitSoft    = uint8(1)
	BindwidthLimitDynamic = uint8(2)
)
