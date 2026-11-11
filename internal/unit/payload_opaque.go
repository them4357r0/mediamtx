package unit

// PayloadOpaque is the payload of an opaque MPEG-TS data track
// (see internal/protocols/mpegts), i.e. a PES payload or a complete private section.
// Data must have at least 1 byte.
type PayloadOpaque struct {
	// stream_id of the PES, 0 for sections.
	StreamID uint8
	Data     []byte
}

func (*PayloadOpaque) isPayload() {}
