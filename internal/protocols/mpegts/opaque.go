package mpegts

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/bluenviron/gortsplib/v5/pkg/format"
	tscodecs "github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts/codecs"
)

// OpaqueRTPMapPrefix is the prefix of the rtpmap of formats that carry opaque MPEG-TS data tracks.
// These tracks (timed ID3, SCTE-35) have no RTP representation: they are only
// forwarded between MPEG-TS publishers and MPEG-TS readers.
const OpaqueRTPMapPrefix = "x-mpegts-opaque/"

// language is carried in the media ID, since gortsplib medias have no language field.
const langIDPrefix = "tslang"

func newOpaqueFormat(codec *tscodecs.Opaque, programDescs []byte) (*format.Generic, error) {
	forma := &format.Generic{
		PayloadTyp: 96,
		RTPMa:      OpaqueRTPMapPrefix + "90000",
		FMT: map[string]string{
			"st":   strconv.Itoa(int(codec.StreamType)),
			"desc": hex.EncodeToString(codec.Descriptors),
			"sec":  strconv.FormatBool(codec.Sections),
		},
	}
	if len(programDescs) != 0 {
		forma.FMT["prog"] = hex.EncodeToString(programDescs)
	}
	err := forma.Init()
	return forma, err
}

// IsOpaqueFormat checks whether the format carries an opaque MPEG-TS data track.
func IsOpaqueFormat(forma format.Format) bool {
	g, ok := forma.(*format.Generic)
	return ok && strings.HasPrefix(strings.ToLower(g.RTPMa), OpaqueRTPMapPrefix)
}

// the fmtp of a format comes from the publisher (SDP): only the stream types that
// are carried as opaque tracks are accepted, and the descriptors are bounded.
const (
	streamTypeID3    = 0x15
	streamTypeSCTE35 = 0x86
	maxOpaqueDescLen = 1024 // bytes, for each of desc and prog
)

func decodeBoundedHex(name string, s string) ([]byte, error) {
	if len(s) > 2*maxOpaqueDescLen {
		return nil, fmt.Errorf("%s is too long (%d bytes, max %d)", name, len(s)/2, maxOpaqueDescLen)
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", name, err)
	}
	return b, nil
}

func opaqueCodecFromFormat(g *format.Generic) (*tscodecs.Opaque, []byte, error) {
	st, err := strconv.ParseUint(g.FMT["st"], 10, 8)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid stream type: %w", err)
	}
	if st != streamTypeID3 && st != streamTypeSCTE35 {
		return nil, nil, fmt.Errorf("unsupported opaque stream type 0x%02x", st)
	}
	desc, err := decodeBoundedHex("desc", g.FMT["desc"])
	if err != nil {
		return nil, nil, err
	}
	prog, err := decodeBoundedHex("prog", g.FMT["prog"])
	if err != nil {
		return nil, nil, err
	}
	return &tscodecs.Opaque{
		StreamType:  uint8(st),
		Descriptors: desc,
		Sections:    g.FMT["sec"] == "true",
	}, prog, nil
}

func languageToID(lang string, n int) string {
	if len(lang) != 3 {
		return ""
	}
	for _, c := range lang {
		if c < 'a' || c > 'z' {
			return ""
		}
	}
	return fmt.Sprintf("%s%s%d", langIDPrefix, lang, n)
}

func idToLanguage(id string) string {
	if len(id) > len(langIDPrefix)+3 && strings.HasPrefix(id, langIDPrefix) {
		return id[len(langIDPrefix) : len(langIDPrefix)+3]
	}
	return ""
}
