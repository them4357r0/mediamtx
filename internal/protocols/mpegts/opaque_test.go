package mpegts

import (
	"strings"
	"testing"

	"github.com/bluenviron/gortsplib/v5/pkg/format"
	tscodecs "github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts/codecs"
	"github.com/stretchr/testify/require"
)

func TestOpaqueFormatRoundTrip(t *testing.T) {
	codec := &tscodecs.Opaque{
		StreamType:  0x86,
		Descriptors: []byte{0x05, 0x04, 'C', 'U', 'E', 'I'},
		Sections:    true,
	}
	prog := []byte{0x05, 0x04, 'C', 'U', 'E', 'I'}

	forma, err := newOpaqueFormat(codec, prog)
	require.NoError(t, err)
	require.True(t, IsOpaqueFormat(forma))
	require.Equal(t, 90000, forma.ClockRate())

	codec2, prog2, err := opaqueCodecFromFormat(forma)
	require.NoError(t, err)
	require.Equal(t, codec, codec2)
	require.Equal(t, prog, prog2)

	forma, err = newOpaqueFormat(&tscodecs.Opaque{StreamType: 0x15}, nil)
	require.NoError(t, err)
	_, prog2, err = opaqueCodecFromFormat(forma)
	require.NoError(t, err)
	require.Empty(t, prog2)
}

func TestLanguageID(t *testing.T) {
	require.Equal(t, "tslangeng1", languageToID("eng", 1))
	require.Equal(t, "eng", idToLanguage("tslangeng1"))
	require.Equal(t, "", languageToID("", 1))
	require.Equal(t, "", languageToID("e1g", 1))
	require.Equal(t, "", idToLanguage("video"))
}

func TestOpaqueCodecFromFormatRejectsUntrustedFmtp(t *testing.T) {
	mk := func(st, desc, prog string) *format.Generic {
		g := &format.Generic{
			PayloadTyp: 96,
			RTPMa:      OpaqueRTPMapPrefix + "90000",
			FMT:        map[string]string{"st": st, "desc": desc, "sec": "true"},
		}
		if prog != "" {
			g.FMT["prog"] = prog
		}
		require.NoError(t, g.Init())
		return g
	}

	for _, st := range []string{"21", "134"} {
		_, _, err := opaqueCodecFromFormat(mk(st, "", ""))
		require.NoError(t, err, st)
	}

	// other stream types, including the ones that carry A/V or are reserved
	for _, st := range []string{"0", "2", "27", "6", "133", "135", "255"} {
		_, _, err := opaqueCodecFromFormat(mk(st, "", ""))
		require.ErrorContains(t, err, "unsupported opaque stream type", st)
	}
	_, _, err := opaqueCodecFromFormat(mk("256", "", ""))
	require.Error(t, err)

	// size caps: exactly 1 KiB is accepted, one more byte is not
	ok := strings.Repeat("ab", 1024)
	big := strings.Repeat("ab", 1025)
	_, _, err = opaqueCodecFromFormat(mk("134", ok, ok))
	require.NoError(t, err)
	_, _, err = opaqueCodecFromFormat(mk("134", big, ""))
	require.ErrorContains(t, err, "desc is too long")
	_, _, err = opaqueCodecFromFormat(mk("134", "", big))
	require.ErrorContains(t, err, "prog is too long")
	_, _, err = opaqueCodecFromFormat(mk("134", "zz", ""))
	require.ErrorContains(t, err, "invalid desc")
}
