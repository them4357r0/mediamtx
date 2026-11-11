package unit

// PayloadOpaque is the payload of an opaque MPEG-TS data track
// (see internal/protocols/mpegts), i.e. a PES payload or a complete private section.
// This must have at least 1 byte.
type PayloadOpaque []byte

func (PayloadOpaque) isPayload() {}
