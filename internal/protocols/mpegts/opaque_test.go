package mpegts

import (
	"testing"

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
