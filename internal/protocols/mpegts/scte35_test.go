package mpegts

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

// buildSCTE35 builds a time_signal splice_info_section.
func buildSCTE35(encrypted bool, ptsAdjustment int64, spliceTime int64) []byte {
	b := []byte{scte35TableID, 0x30, 0}
	b = append(b, 0) // protocol_version
	adj := make([]byte, 5)
	adj[0] = byte(ptsAdjustment >> 32 & 1)
	if encrypted {
		adj[0] |= 0x80 | 1<<1 // encrypted_packet + algorithm 1
	}
	binary.BigEndian.PutUint32(adj[1:], uint32(ptsAdjustment))
	b = append(b, adj...)
	b = append(b, 0, 0xff, 0xf0|0, 5) // cw_index, tier(12)=0xfff, splice_command_length=5
	b = append(b, 0x06)               // time_signal
	st := make([]byte, 5)
	st[0] = 0x80 | 0x7e | byte(spliceTime>>32&1)
	binary.BigEndian.PutUint32(st[1:], uint32(spliceTime))
	b = append(b, st...)
	b = append(b, 0, 0) // descriptor_loop_length
	if encrypted {
		b = append(b, 0xde, 0xad, 0xbe, 0xef) // E_CRC_32 (opaque)
	}
	b = append(b, 0, 0, 0, 0) // CRC_32
	sl := len(b) - 3
	b[1] = 0x30 | byte(sl>>8)
	b[2] = byte(sl)
	binary.BigEndian.PutUint32(b[len(b)-4:], crc32MPEG(b[:len(b)-4]))
	return b
}

func scte35Adjustment(b []byte) int64 {
	return int64(b[4]&1)<<32 | int64(binary.BigEndian.Uint32(b[5:9]))
}

func TestRebaseSCTE35(t *testing.T) {
	for _, ca := range []struct {
		name      string
		encrypted bool
		adj       int64
		k         int64
		expected  int64
	}{
		{"simple", false, 1000, 253920, (1000 - 253920) & mask33},
		{"wrap below zero", false, 5, 10, mask33 + 1 - 5},
		{"adjustment bit 32", false, 1<<32 + 7, 1<<32 + 2, 5},
		{"k zero", false, 123, 0, 123},
		{"k above 2^33 is wrapped by the caller", false, 100, 90, 10},
		{"encrypted", true, 1000, 253920, (1000 - 253920) & mask33},
	} {
		t.Run(ca.name, func(t *testing.T) {
			in := buildSCTE35(ca.encrypted, ca.adj, 5000)
			orig := append([]byte(nil), in...)

			out := rebaseSCTE35(in, ca.k)

			require.Equal(t, orig, in, "input must not be modified")
			require.Equal(t, len(in), len(out))
			require.Equal(t, ca.expected, scte35Adjustment(out))
			require.Zero(t, crc32MPEG(out), "outer CRC_32 must be valid")

			// effective splice time moves by -k, modulo 2^33
			eff := func(b []byte) int64 {
				st := int64(b[14]&1)<<32 | int64(binary.BigEndian.Uint32(b[15:19]))
				return (st + scte35Adjustment(b)) & mask33
			}
			require.Equal(t, (eff(in)-ca.k)&mask33, eff(out))

			// everything but pts_adjustment bits and the outer CRC is identical
			// (including the encrypted flag, and E_CRC_32 of encrypted sections)
			a, b := append([]byte(nil), in...), append([]byte(nil), out...)
			for _, x := range [][]byte{a, b} {
				x[4] &= 0xFE
				for i := 5; i < 9; i++ {
					x[i] = 0
				}
				for i := len(x) - 4; i < len(x); i++ {
					x[i] = 0
				}
			}
			require.Equal(t, a, b)
			require.Equal(t, ca.encrypted, out[4]&0x80 != 0)
		})
	}

	t.Run("unrecognized sections are untouched", func(t *testing.T) {
		in := buildSCTE35(false, 1, 1)
		bad := append([]byte(nil), in...)
		bad[0] = 0x00
		require.Equal(t, bad, rebaseSCTE35(bad, 5))
		require.Equal(t, in[:10], rebaseSCTE35(in[:10], 5))
		trunc := append(append([]byte(nil), in...), 1)
		require.Equal(t, trunc, rebaseSCTE35(trunc, 5))
	})
}
