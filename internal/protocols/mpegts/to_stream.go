// Package mpegts contains MPEG-ts utilities.
package mpegts

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/bluenviron/gortsplib/v5/pkg/description"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts"
	tscodecs "github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts/codecs"

	"github.com/bluenviron/mediamtx/internal/logger"
	"github.com/bluenviron/mediamtx/internal/stream"
	"github.com/bluenviron/mediamtx/internal/unit"
)

var errNoSupportedCodecs = errors.New(
	"the stream doesn't contain any supported codec, which are currently " +
		"H265, H264, MPEG-4 Video, MPEG-1/2 Video, Opus, MPEG-4 Audio, MPEG-1/2 Audio, AC-3")

// ToStream maps a MPEG-TS stream to a MediaMTX stream.
func ToStream(
	r *EnhancedReader,
	subStream **stream.SubStream,
	l logger.Writer,
) ([]*description.Media, error) {
	var medias []*description.Media //nolint:prealloc
	var unsupportedTracks []int

	tdi := &mpegts.TimeDecoder{}
	tdi.Initialize()

	td := &tsClock{TimeDecoder: tdi}
	ob := &opaqueBuffer{l: l}
	td.onInit = ob.flush

	programDescs := r.ProgramDescriptors()

	for i, track := range r.Tracks() { //nolint:dupl
		var medi *description.Media

		switch codec := track.Codec.(type) {
		case *tscodecs.H265:
			medi = &description.Media{
				Type: description.MediaTypeVideo,
				Formats: []format.Format{&format.H265{
					PayloadTyp: 96,
				}},
			}

			r.OnDataH265(track, func(pts int64, _ int64, au [][]byte) error {
				pts = td.Decode(pts)

				(*subStream).WriteUnit(medi, medi.Formats[0], &unit.Unit{
					PTS:     pts, // no conversion is needed since clock rate is 90khz in both MPEG-TS and RTSP
					Payload: unit.PayloadH265(au),
				})
				return nil
			})

		case *tscodecs.H264:
			medi = &description.Media{
				Type: description.MediaTypeVideo,
				Formats: []format.Format{&format.H264{
					PayloadTyp:        96,
					PacketizationMode: 1,
				}},
			}

			r.OnDataH264(track, func(pts int64, _ int64, au [][]byte) error {
				pts = td.Decode(pts)

				(*subStream).WriteUnit(medi, medi.Formats[0], &unit.Unit{
					PTS:     pts, // no conversion is needed since clock rate is 90khz in both MPEG-TS and RTSP
					Payload: unit.PayloadH264(au),
				})
				return nil
			})

		case *tscodecs.MPEG4Video:
			medi = &description.Media{
				Type: description.MediaTypeVideo,
				Formats: []format.Format{&format.MPEG4Video{
					PayloadTyp: 96,
				}},
			}

			r.OnDataMPEG4Video(track, func(pts int64, frame []byte) error {
				pts = td.Decode(pts)

				(*subStream).WriteUnit(medi, medi.Formats[0], &unit.Unit{
					PTS:     pts, // no conversion is needed since clock rate is 90khz in both MPEG-TS and RTSP
					Payload: unit.PayloadMPEG4Video(frame),
				})
				return nil
			})

		case *tscodecs.MPEG1Video:
			medi = &description.Media{
				Type:    description.MediaTypeVideo,
				Formats: []format.Format{&format.MPEG1Video{}},
			}

			r.OnDataMPEG1Video(track, func(pts int64, frame []byte) error {
				pts = td.Decode(pts)

				(*subStream).WriteUnit(medi, medi.Formats[0], &unit.Unit{
					PTS:     pts, // no conversion is needed since clock rate is 90khz in both MPEG-TS and RTSP
					Payload: unit.PayloadMPEG1Video(frame),
				})
				return nil
			})

		case *tscodecs.Opus:
			medi = &description.Media{
				Type: description.MediaTypeAudio,
				Formats: []format.Format{&format.Opus{
					PayloadTyp:   96,
					ChannelCount: codec.Desc.ChannelCount(),
				}},
			}

			r.OnDataOpus(track, func(pts int64, packets [][]byte) error {
				pts = td.Decode(pts)

				(*subStream).WriteUnit(medi, medi.Formats[0], &unit.Unit{
					PTS:     multiplyAndDivide(pts, int64(medi.Formats[0].ClockRate()), 90000),
					Payload: unit.PayloadOpus(packets),
				})
				return nil
			})

		case *tscodecs.KLV:
			medi = &description.Media{
				Type: description.MediaTypeApplication,
				Formats: []format.Format{&format.KLV{
					PayloadTyp: 96,
				}},
			}
			r.OnDataKLV(track, func(pts int64, uni []byte) error {
				pts = td.Decode(pts)

				(*subStream).WriteUnit(medi, medi.Formats[0], &unit.Unit{
					PTS:     pts,
					Payload: unit.PayloadKLV(uni),
				})
				return nil
			})

		case *tscodecs.MPEG4Audio:
			medi = &description.Media{
				Type: description.MediaTypeAudio,
				Formats: []format.Format{&format.MPEG4Audio{
					PayloadTyp:       96,
					SizeLength:       13,
					IndexLength:      3,
					IndexDeltaLength: 3,
					Config:           &codec.Config,
				}},
			}

			r.OnDataMPEG4Audio(track, func(pts int64, aus [][]byte) error {
				pts = td.Decode(pts)

				(*subStream).WriteUnit(medi, medi.Formats[0], &unit.Unit{
					PTS:     multiplyAndDivide(pts, int64(medi.Formats[0].ClockRate()), 90000),
					Payload: unit.PayloadMPEG4Audio(aus),
				})
				return nil
			})

		case *tscodecs.MPEG4AudioLATM:
			// We are dealing with a LATM stream with in-band configuration.
			// Although in theory this can be streamed with RTSP (RFC6416 with cpresent=1),
			// in practice there is no player that supports it.
			// Therefore, convert the stream to a LATM stream with out-of-band configuration.
			streamMuxConfig := r.latmConfigs[track.PID]
			medi = &description.Media{
				Type: description.MediaTypeAudio,
				Formats: []format.Format{&format.MPEG4AudioLATM{
					PayloadTyp:      96,
					CPresent:        false,
					ProfileLevelID:  30,
					StreamMuxConfig: streamMuxConfig,
				}},
			}
			clockRate := medi.Formats[0].ClockRate()

			r.OnDataMPEG4AudioLATM(track, func(pts int64, els [][]byte) error {
				pts = td.Decode(pts)

				pts = multiplyAndDivide(pts, int64(clockRate), 90000)

				for _, el := range els {
					var elIn mpeg4audio.AudioMuxElement
					elIn.MuxConfigPresent = true
					elIn.StreamMuxConfig = streamMuxConfig
					err := elIn.Unmarshal(el)
					if err != nil {
						return err
					}

					if !reflect.DeepEqual(elIn.StreamMuxConfig, streamMuxConfig) {
						return fmt.Errorf("dynamic stream mux config is not supported")
					}

					var elOut mpeg4audio.AudioMuxElement
					elOut.MuxConfigPresent = false
					elOut.StreamMuxConfig = streamMuxConfig
					elOut.Payloads = elIn.Payloads
					buf, err := elOut.Marshal()
					if err != nil {
						return err
					}

					(*subStream).WriteUnit(medi, medi.Formats[0], &unit.Unit{
						PTS:     pts,
						Payload: unit.PayloadMPEG4AudioLATM(buf),
					})

					pts += mpeg4audio.SamplesPerAccessUnit
				}

				return nil
			})

		case *tscodecs.MPEG1Audio:
			medi = &description.Media{
				Type:    description.MediaTypeAudio,
				Formats: []format.Format{&format.MPEG1Audio{}},
			}

			r.OnDataMPEG1Audio(track, func(pts int64, frames [][]byte) error {
				pts = td.Decode(pts)

				(*subStream).WriteUnit(medi, medi.Formats[0], &unit.Unit{
					PTS:     pts, // no conversion is needed since clock rate is 90khz in both MPEG-TS and RTSP
					Payload: unit.PayloadMPEG1Audio(frames),
				})
				return nil
			})

		case *tscodecs.AC3:
			medi = &description.Media{
				Type: description.MediaTypeAudio,
				Formats: []format.Format{&format.AC3{
					PayloadTyp:   96,
					SampleRate:   codec.SampleRate,
					ChannelCount: codec.ChannelCount,
				}},
			}

			r.OnDataAC3(track, func(pts int64, frame []byte) error {
				pts = td.Decode(pts)

				(*subStream).WriteUnit(medi, medi.Formats[0], &unit.Unit{
					PTS:     multiplyAndDivide(pts, int64(medi.Formats[0].ClockRate()), 90000),
					Payload: unit.PayloadAC3{frame},
				})
				return nil
			})

		case *tscodecs.Opaque:
			forma, err := newOpaqueFormat(codec, programDescs)
			if err != nil {
				return nil, err
			}

			medi = &description.Media{
				Type:    description.MediaTypeApplication,
				Formats: []format.Format{forma},
			}

			r.OnDataOpaque(track, func(pts int64, streamID uint8, data []byte) error {
				write := func() {
					// time base is moved like for other tracks (units older than the first
					// decoded sample get a negative PTS, that is kept)
					dec := tdi.Decode(pts)
					if dec < 0 {
						ob.negative(dec)
					}

					if codec.Sections && codec.StreamType == scte35StreamType {
						// pts_adjustment is moved by the same amount
						data = rebaseSCTE35(data, (pts-dec)&mask33)
					}

					(*subStream).WriteUnit(medi, medi.Formats[0], &unit.Unit{
						PTS:     dec,
						Payload: &unit.PayloadOpaque{StreamID: streamID, Data: data},
					})
				}

				// opaque tracks do not establish the time base: until a decoded track
				// has done it, units are kept
				if !tdi.Initialized() {
					data = append([]byte(nil), data...)
					ob.add(pts, len(data), write)
					return nil
				}

				write()
				return nil
			})

		default:
			unsupportedTracks = append(unsupportedTracks, i+1)
			continue
		}

		if medi.Type == description.MediaTypeAudio {
			medi.ID = languageToID(track.Language, len(medias))
		}

		medias = append(medias, medi)
	}

	if len(medias) == 0 {
		return nil, errNoSupportedCodecs
	}

	for _, id := range unsupportedTracks {
		l.Log(logger.Warn, "skipping track %d (unsupported codec)", id)
	}

	return medias, nil
}

const scte35StreamType = 0x86

// tsClock is a TimeDecoder that notifies when it has been used for the first time.
type tsClock struct {
	*mpegts.TimeDecoder
	onInit func()
}

// Decode decodes a MPEG-TS timestamp.
func (c *tsClock) Decode(ts int64) int64 {
	v := c.TimeDecoder.Decode(ts)
	if c.onInit != nil {
		f := c.onInit
		c.onInit = nil
		f()
	}
	return v
}

const (
	opaqueBufferMaxUnits = 64
	opaqueBufferMaxBytes = 256 * 1024
	opaqueBufferMaxSpan  = 2 * 90000
)

// opaqueBuffer keeps opaque units that are received before the time base is established.
type opaqueBuffer struct {
	l logger.Writer

	units     []func()
	bytes     int
	firstPTS  int64
	dropped   int
	negLogged bool
}

func (b *opaqueBuffer) add(pts int64, size int, write func()) {
	if len(b.units) == 0 {
		b.firstPTS = pts
	}

	if len(b.units) >= opaqueBufferMaxUnits ||
		b.bytes+size > opaqueBufferMaxBytes ||
		(pts-b.firstPTS)&mask33 > opaqueBufferMaxSpan {
		b.dropped++
		return
	}

	b.units = append(b.units, write)
	b.bytes += size
}

func (b *opaqueBuffer) flush() {
	units := b.units
	b.units = nil
	b.bytes = 0

	for _, w := range units {
		w()
	}

	if b.dropped != 0 {
		b.l.Log(logger.Warn, "dropped %d timed metadata unit(s) received before the first audio/video sample "+
			"(buffer limits: %d units, %d bytes, %d s)",
			b.dropped, opaqueBufferMaxUnits, opaqueBufferMaxBytes, opaqueBufferMaxSpan/90000)
	}
}

// negative reports an opaque unit that is older than the first decoded audio/video sample.
// It is kept with a negative PTS, that is written as a 33-bit wrap.
func (b *opaqueBuffer) negative(dec int64) {
	if !b.negLogged {
		b.negLogged = true
		b.l.Log(logger.Info, "timed metadata unit older than the first audio/video sample "+
			"(%d ticks): kept with a negative PTS", dec)
	}
}
