package mpegts

import (
	"encoding/binary"
)

const (
	scte35TableID = 0xFC
	mask33        = 0x1FFFFFFFF
)

var crc32MPEGTable = func() [256]uint32 {
	var t [256]uint32
	for i := range t {
		c := uint32(i) << 24
		for range 8 {
			if c&0x80000000 != 0 {
				c = (c << 1) ^ 0x04C11DB7
			} else {
				c <<= 1
			}
		}
		t[i] = c
	}
	return t
}()

func crc32MPEG(b []byte) uint32 {
	c := uint32(0xFFFFFFFF)
	for _, v := range b {
		c = (c << 8) ^ crc32MPEGTable[byte(c>>24)^v]
	}
	return c
}

// rebaseSCTE35 returns a copy of a splice_info_section in which pts_adjustment is decreased by k
// (modulo 2^33) and CRC_32 is recomputed. The effective splice time of the section
// (splice_time + pts_adjustment) therefore moves by -k, like the PTS of the other tracks
// when their time base is moved by k.
// Everything else is untouched; in particular, in encrypted sections, the encrypted part and E_CRC_32
// are not modified (pts_adjustment and the outer CRC_32 are never encrypted).
// Sections that are not recognized are returned as they are.
func rebaseSCTE35(sec []byte, k int64) []byte {
	// table_id(1) + length(2) + protocol_version(1) + enc/pts_adjustment(5) + ... + CRC_32(4)
	if len(sec) < 9+4 || sec[0] != scte35TableID ||
		3+int(sec[1]&0x0F)<<8+int(sec[2]) != len(sec) {
		return sec
	}

	adj := int64(sec[4]&1)<<32 | int64(binary.BigEndian.Uint32(sec[5:9]))
	adj = (adj - k) & mask33

	out := append([]byte(nil), sec...)
	out[4] = out[4]&0xFE | byte(adj>>32)
	binary.BigEndian.PutUint32(out[5:9], uint32(adj))
	binary.BigEndian.PutUint32(out[len(out)-4:], crc32MPEG(out[:len(out)-4]))
	return out
}
