package mpegts

import (
	"bufio"
	"bytes"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/bluenviron/gortsplib/v5/pkg/description"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
	mcmpegts "github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts"
	tscodecs "github.com/bluenviron/mediacommon/v2/pkg/formats/mpegts/codecs"
	srt "github.com/datarhei/gosrt"
	"github.com/stretchr/testify/require"

	"github.com/bluenviron/mediamtx/internal/logger"
	"github.com/bluenviron/mediamtx/internal/stream"
	"github.com/bluenviron/mediamtx/internal/test"
	"github.com/bluenviron/mediamtx/internal/unit"
)

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buf.Bytes()...)
}

type stubConn struct{ srt.Conn }

func (stubConn) SetWriteDeadline(time.Time) error { return nil }

var (
	rtID3Descs = []byte{
		0x26, 0x0d, 0xff, 0xff, 0x49, 0x44, 0x33, 0x20,
		0xff, 0x00, 0x0f, 0x00, 0x00, 0x00, 0x00,
	}
	rtCUEI = []byte{0x05, 0x04, 'C', 'U', 'E', 'I'}
	rtSPS  = []byte{
		0x67, 0x42, 0xc0, 0x28, 0xd9, 0x00, 0x78, 0x02,
		0x27, 0xe5, 0x84, 0x00, 0x00, 0x03, 0x00, 0x04,
		0x00, 0x00, 0x03, 0x00, 0xf0, 0x3c, 0x60, 0xc9, 0x20,
	}
)

func rtTS(t *testing.T) (ts []byte, scte []byte) {
	video := &mcmpegts.Track{PID: 256, Codec: &tscodecs.H264{}}
	audio := &mcmpegts.Track{
		PID: 257, Language: "eng",
		Codec: &tscodecs.MPEG4Audio{Config: mpeg4audio.AudioSpecificConfig{
			Type: 2, SampleRate: 48000, ChannelConfig: 2, ChannelCount: 2, //nolint:staticcheck
		}},
	}
	id3 := &mcmpegts.Track{PID: 258, Codec: &tscodecs.Opaque{StreamType: 0x15, Descriptors: rtID3Descs}}
	sc := &mcmpegts.Track{PID: 496, Codec: &tscodecs.Opaque{StreamType: 0x86, Descriptors: rtCUEI, Sections: true}}

	var buf bytes.Buffer
	w := &mcmpegts.Writer{W: &buf, Tracks: []*mcmpegts.Track{video, audio, id3, sc}, ProgramDescriptors: rtCUEI}
	require.NoError(t, w.Initialize())

	scte = buildSCTE35(false, 90000, 100000+45000)
	aac := []byte{0x21, 0x10, 0x04, 0x60, 0x8c, 0x1c}
	au := [][]byte{rtSPS, {0x68, 0xce, 0x3c, 0x80}, {byte(h264.NALUTypeIDR), 1, 2}}

	// the ID3 unit (PTS older than the first video frame) arrives before any frame is decoded
	require.NoError(t, w.WriteH264(video, 100000, 100000, au))
	require.NoError(t, w.WriteOpaque(id3, 98000, 0xFC, []byte("ID3-a")))
	require.NoError(t, w.WriteOpaque(sc, 0, 0, scte))
	require.NoError(t, w.WriteH264(video, 103000, 103000, au))
	require.NoError(t, w.WriteMPEG4Audio(audio, 103000, [][]byte{aac}))
	require.NoError(t, w.WriteOpaque(id3, 103000, 0xBD, []byte("ID3-b")))
	require.NoError(t, w.WriteH264(video, 106000, 106000, au))
	require.NoError(t, w.WriteMPEG4Audio(audio, 106000, [][]byte{aac}))
	require.NoError(t, w.WriteH264(video, 109000, 109000, au))
	require.NoError(t, w.WriteMPEG4Audio(audio, 109000, [][]byte{aac}))
	return buf.Bytes(), scte
}

func TestOpaqueToStreamFromStream(t *testing.T) {
	ts, scte := rtTS(t)

	var logs []string
	lg := test.Logger(func(_ logger.Level, f string, args ...any) {
		logs = append(logs, f)
	})

	r := &EnhancedReader{R: bytes.NewReader(ts)}
	require.NoError(t, r.Initialize())

	var subStream *stream.SubStream
	medias, err := ToStream(r, &subStream, lg)
	require.NoError(t, err)
	require.Len(t, medias, 4)
	require.Equal(t, "tslangeng1", medias[1].ID)
	require.True(t, IsOpaqueFormat(medias[2].Formats[0]))

	desc := &description.Session{Medias: medias}
	strm := &stream.Stream{OrigDesc: desc, WriteQueueSize: 512, RTPMaxPayloadSize: 1450}
	require.NoError(t, strm.Initialize())
	defer strm.Close()

	subStream = &stream.SubStream{Stream: strm}
	require.NoError(t, subStream.Initialize())

	// an RTP-based reader (like RTSP) keeps working and does not see opaque tracks
	rtpUnits := make(chan *unit.Unit, 100)
	rtpReader := &stream.Reader{Parent: lg}
	rtpReader.OnData(medias[0], medias[0].Formats[0], func(u *unit.Unit) error {
		rtpUnits <- u
		return nil
	})
	strm.AddReader(rtpReader)
	defer strm.RemoveReader(rtpReader)

	// MPEG-TS reader
	out := &lockedBuffer{}
	bw := bufio.NewWriter(out)
	tsReader := &stream.Reader{Parent: lg}
	require.NoError(t, FromStream(desc, tsReader, bw, stubConn{}, time.Second))
	strm.AddReader(tsReader)
	defer strm.RemoveReader(tsReader)

	for {
		err = r.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	require.Eventually(t, func() bool {
		return bytes.Contains(out.bytes(), []byte("ID3-b"))
	}, 5*time.Second, 10*time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	select {
	case u := <-rtpUnits:
		require.NotEmpty(t, u.RTPPackets)
	default:
		t.Fatal("RTP reader did not receive video")
	}

	// read back
	rr := &mcmpegts.Reader{R: bytes.NewReader(out.bytes())}
	require.NoError(t, rr.Initialize())
	require.Equal(t, rtCUEI, rr.ProgramDescriptors())

	tracks := rr.Tracks()
	require.Len(t, tracks, 4)
	require.Equal(t, "eng", tracks[1].Language)
	require.Equal(t, &tscodecs.Opaque{StreamType: 0x15, Descriptors: rtID3Descs}, tracks[2].Codec)
	require.Equal(t, &tscodecs.Opaque{StreamType: 0x86, Descriptors: rtCUEI, Sections: true}, tracks[3].Codec)

	type got struct {
		pts int64
		sid uint8
		d   string
	}
	var id3 []got
	var sections [][]byte

	rr.OnDataH264(tracks[0], func(int64, int64, [][]byte) error { return nil })
	rr.OnDataMPEG4Audio(tracks[1], func(int64, [][]byte) error { return nil })
	rr.OnDataOpaque(tracks[2], func(pts int64, sid uint8, data []byte) error {
		id3 = append(id3, got{pts, sid, string(data)})
		return nil
	})
	rr.OnDataOpaque(tracks[3], func(_ int64, _ uint8, data []byte) error {
		sections = append(sections, data)
		return nil
	})
	for {
		err = rr.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	// K = first decoded video PTS (100000). The first ID3 unit was received before
	// the time base existed and is older than it: kept, negative, written as 33-bit wrap.
	require.Equal(t, []got{
		{mask33 + 1 - 2000, 0xFC, "ID3-a"},
		{3000, 0xBD, "ID3-b"},
	}, id3)

	require.Len(t, sections, 1)
	require.Equal(t, rebaseSCTE35(scte, 100000), sections[0])
	require.Zero(t, crc32MPEG(sections[0]))
	require.Equal(t, int64(90000-100000)&mask33, scte35Adjustment(sections[0]))

	require.Contains(t, logs, "timed metadata unit older than the first audio/video sample "+
		"(%d ticks): kept with a negative PTS")
}
