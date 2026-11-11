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

func opaqueCodecFromFormat(g *format.Generic) (*tscodecs.Opaque, []byte, error) {
	st, err := strconv.ParseUint(g.FMT["st"], 10, 8)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid stream type: %w", err)
	}
	desc, err := hex.DecodeString(g.FMT["desc"])
	if err != nil {
		return nil, nil, err
	}
	prog, err := hex.DecodeString(g.FMT["prog"])
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
