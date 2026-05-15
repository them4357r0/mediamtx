

vlc --network-caching=50 rtsp://...


git clone https://github.com/bluenviron/mediamtx
cd mediamtx
/mnt/data/MediaMTX/go-1.25.1/bin/go generate ./...
CGO_ENABLED=0 ../go-1.25.1/bin/go  build .


# Windows 64비트 (amd64) 대상 빌드
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 /mnt/data/MediaMTX/go-1.25.1/bin/go build -o mediamtx.exe .


MediaMTX API

Configuration	
    Global	        GET	    http://61.96.224.183:9997/v3/config/global/get
		            PATCH	http://61.96.224.183:9997/v3/config/global/patch
			
	Default Path	GET	    http://61.96.224.183:9997/v3/config/pathdefaults/get
		            PATCH	http://61.96.224.183:9997/v3/config/pathdefaults/patch
			
	All Paths	    GET	    http://61.96.224.183:9997/v3/config/paths/list
	Path	        GET	    http://61.96.224.183:9997/v3/config/paths/get/{name}
	Add Path	    POST	http://61.96.224.183:9997/v3/config/paths/add/{name}
	Patch Path	    PATCH	http://61.96.224.183:9997/v3/config/paths/patch/{name}
	Replace Path	POST	http://61.96.224.183:9997/v3/config/paths/replace/{name}
	Remove Path	    DELETE	http://61.96.224.183:9997/v3/config/paths/delete/{name}
			


Path	All Paths	GET	    http://61.96.224.183:9997/v3/paths/list
	    Path	    GET	    http://61.96.224.183:9997/v3/paths/get/{name}
                            http://61.96.224.183:9997/v3/paths/get/livex025/origin
			
HLS	    All Muxers	GET	    http://61.96.224.183:9997/v3/hlsmuxers/list
	    Muxer	    GET	    http://61.96.224.183:9997/v3/hlsmuxers/get/{name}
                            http://61.96.224.183:9997/v3/hlsmuxers/get/livex025/origin
                            
			
RTSP	All Connections	    GET	http://61.96.224.183:9997/v3/rtspconns/list
	    Connection	        GET	http://61.96.224.183:9997/v3/rtspconns/get/{id}
	    All Session	        GET	http://61.96.224.183:9997/v3/rtspsessions/list
	    Session	            GET	http://61.96.224.183:9997/v3/rtspsessions/get/{id}
	    Kicks out Session	POST	http://61.96.224.183:9997/v3/rtspsessions/kick/{id}
			
Recordings	All	            GET	http://61.96.224.183:9997/v3/recordings/list
	        a Path	        GET	http://61.96.224.183:9997/v3/recordings/get/{name}
                                http://61.96.224.183:9997/v3/recordings/get/livex025/record
	        Delete Segment	DELETE	http://61.96.224.183:9997/v3/recordings/deletesegment



# Playback API
http://localhost:9996/list?path={mypath}&start={start}&end={end}
http://localhost:9996/list?path={mypath}
http://61.96.224.183:9996/list?path=livex025/record

http://localhost:9996/get?path={mypath}&start={start}&duration={duration}&format={format}
http://61.96.224.183:9996/get?path=livex025/record&start=2025-07-24T06:41:45.41678Z&duration=10




{
"name": "string",
"source": "string",		# (publisher), rtsp-url, hls-url
"sourceOnDemand": true,

"record": true,

"rtspTransport": "string",		# (automatic), udp, multicast, tcp.
}


{
"name": "string",
"source": "string",
"sourceFingerprint": "string",
"sourceOnDemand": true,
"sourceOnDemandStartTimeout": "string",
"sourceOnDemandCloseAfter": "string",
"maxReaders": 0,
"srtReadPassphrase": "string",
"fallback": "string",
"useAbsoluteTimestamp": true,
"record": true,
"recordPath": "string",
"recordFormat": "string",
"recordPartDuration": "string",
"recordMaxPartSize": "string",
"recordSegmentDuration": "string",
"recordDeleteAfter": "string",
"overridePublisher": true,
"srtPublishPassphrase": "string",
"rtspTransport": "string",
"rtspAnyPort": true,
"rtspRangeType": "string",
"rtspRangeStart": "string",
"sourceRedirect": "string",

"runOnInit": "string",
"runOnInitRestart": true,
"runOnDemand": "string",
"runOnDemandRestart": true,
"runOnDemandStartTimeout": "string",
"runOnDemandCloseAfter": "string",
"runOnUnDemand": "string",
"runOnReady": "string",
"runOnReadyRestart": true,
"runOnNotReady": "string",
"runOnRead": "string",
"runOnReadRestart": true,
"runOnUnread": "string",
"runOnRecordSegmentCreate": "string",
"runOnRecordSegmentComplete": "string"



# golang-go 업데이트 
/data/MediaMTX/go-1.25.1
ln -s /data/MediaMTX/go-1.25.1/bin/go go

# build 
git checkout v1.15.6
./go generate ./...
CGO_ENABLED=0 ./go build .



############################################################
## MediaMTX 소스 수정 사항 (v1.15.6)

############################################################
# udpMaxPayloadSize 최대값 체크 기능 수정 (optional)   ===> (수정 불필요?, 제거 예정)
# rtspMaxPacketSize 설정 추가,  RTSP Sever 설정 기능 추가, 체크 기능 수정  (optional)
/data/MediaMTX/mediamtx-1.15.0/internal
conf/conf.go:239:       RTSPMaxPacketSize     uint             `json:"rtspMaxPacketSize"` // // 2025.08.25 muhwan ADD
conf/conf.go:536:               // 2025.08.25 muhwan MODIFY
core/core.go:399:                       MaxPacketSize:       p.conf.RTSPMaxPacketSize, // 2025.08.25 muhwan ADD
core/core.go:441:                       MaxPacketSize:       p.conf.RTSPMaxPacketSize, // 2025.08.25 muhwan ADD
core/core.go:738:               newConf.RTSPMaxPacketSize != p.conf.RTSPMaxPacketSize || // 2025.08.25 muhwan ADD
core/core.go:762:               newConf.RTSPMaxPacketSize != p.conf.RTSPMaxPacketSize || // 2025.08.25 muhwan ADD
servers/rtsp/server.go:76:      MaxPacketSize       uint // 2025.08.25 muhwan ADD
servers/rtsp/server.go:122:             MaxPacketSize:     int(s.MaxPacketSize), // 2025.08.25 muhwan ADD
gortsplib/v5@v5.0.0/server.go:182:              // 2025.08.25 @muhwan MODIFY


############################################################
# RTSP GET_PARAMETER 응답메시지 수정: Nimble Streamer 호환성 문제로 헤더 제거 (Content-Type: text/parameters)
/home/ktict/go/pkg/mod/github.com/bluenviron

v5@v5.2.2/server_session.go:1642:
                // @muhwan 2025.09.03 MODIFY
                // return &base.Response{
                //      StatusCode: base.StatusOK,
                //      Header: base.Header{
                //              "Content-Type": base.HeaderValue{"text/parameters"},
                //      },
                //      Body: []byte{},
                //}, nil
                return &base.Response{ StatusCode: base.StatusOK }, nil


############################################################
# DTS 기능 수정  
# reordered frames 발생으로 HLS 종료되는 증상 
~/go/pkg/mod/github.com/bluenviron/mediacommon/

v2@v2.6.0/pkg/codecs/h264/dts_extractor.go:142:
        case NALUTypeIDR:
            idr = nalu
            d.reorderedFrames = 0   // 2025.11.07 @muhwan ADD
            break outer

v2@v2.6.0/pkg/codecs/h264/dts_extractor.go:205:
                    if (d.reorderedFrames + increase) > maxReorderedFrames {
                        // 2025.11.10 @muhwan   MODIFY
                        // return 0, false, fmt.Errorf("too many reordered frames (%d)", d.reorderedFrames+increase)
                        fmt.Printf("too many reordered frames-1 (%d)", d.reorderedFrames+increase)
                    }

v2@v2.6.0/pkg/codecs/h264/dts_extractor.go:244:
        if (d.reorderedFrames + increase) > maxReorderedFrames {
            // 2025.11.10 @muhwan   MODIFY
            // return 0, false, fmt.Errorf("too many reordered frames (%d)", d.reorderedFrames+increase)
            fmt.Printf("too many reordered frames-2 (%d)\n", d.reorderedFrames+increase)
        }

v2@v2.6.0/pkg/codecs/h264/dts_extractor.go:256:
        if (d.reorderedFrames + increase) > maxReorderedFrames {
            // 2025.11.10 @muhwan   MODIFY
            // return 0, false, fmt.Errorf("too many reordered frames (%d)", d.reorderedFrames+increase)
            fmt.Printf("too many reordered frames-3 (%d)\n", d.reorderedFrames+increase)
        }

~/go/pkg/mod/github.com/bluenviron/gohlslib/

v2@v2.2.4/muxer_segmenter.go:355:
    if err != nil {
        // 2025.11.10 @muhwan MODIFY
        // return fmt.Errorf("unable to extract DTS: %w", err)
        track.stream.onEncodeError(err)
        return nil
    }



############################################################
# SDP 에러 수정: v 필드 순서, m=meta 처리 (규격오류)
~/go/pkg/mod/github.com/bluenviron/gortsplib/

v5@v5.2.2/pkg/sdp/sdp.go:441:
	if fields[0] != "video" &&
		fields[0] != "audio" &&
		fields[0] != "application" &&
		!strings.HasPrefix(fields[0], "application/") &&
		fields[0] != "metadata" &&
		fields[0] != "meta" && // 2025.11.18	@muhwan	ADD	(for nimble)
		fields[0] != "text" {
		return fmt.Errorf("%w `%v`", errSDPInvalidValue, fields[0])
	}
	newMediaDesc.MediaName.Media = fields[0]

v5@v5.2.2/pkg/sdp/sdp.go:551:
	case 'v': // 2025.11.18	@muhwan	ADD	(for nimble)
		err := s.unmarshalProtocolVersion(val)
		if err != nil {
			return err
		}



############################################################
# RTST session 헤더 중복 오류 수정
~/go/pkg/mod/github.com/bluenviron/gortsplib/

v5@v5.2.2/pkg/headers/session.go:26:
        // 2025.11.18   @muhwan REMOVE
        // if len(v) > 1 {
        //      return fmt.Errorf("value provided multiple times (%v)", v)
        //}


############################################################
# interleaved 파라메터 중복 문제, (포항시 예외처리)
~/go/pkg/mod/github.com/bluenviron/gortsplib/

v5@v5.0.0/pkg/headers/keyval.go:56:
            // 2025.12.23 @muhwan MODIFY, 없는 경우에만 추가 (do not overwrite)
            // 포항시 Transport: RTP/AVP/TCP;unicast;interleaved=0-1;interleaved=0-3
            // ret[k] = v
            if _, exists := ret[k]; exists {
                fmt.Println("# Exist paramemter skip, ", k, "=", v)
            } else {
                ret[k] = v
            }

############################################################
# 외부 인증서버 사용시 HLS 인증요청 부하 문제 수정 
# http request url:  query 문 token 인증 방식 사용, 401->403  응답수정
# authMethod: http
# authHTTPAddress: http://127.0.0.1:8082/mtxapi/auth

/mediamtx-1.15.6/internal/auth/manager.go:  
	// 2026.02.23 @muhwan ADD
	if req.Protocol == ProtocolHLS && req.File != "index.m3u8" {
		// fmt.Println("## HLS files do not auth request ", req.File)
		return nil
	}

    // 2026.02.24 @muhwan MODIFY  query token 인증
    // AskCredentials: (req.Credentials.User == "" && req.Credentials.Pass == "" && req.Credentials.Token == ""),
    AskCredentials: (req.Credentials.User == "" && req.Credentials.Pass == "" && req.Credentials.Token == "" && req.Query == ""),

/mediamtx-1.15.6/internal/auth/request.go:  
	File             string // 2026.02.23 @muhwan ADD

    // 2026.02.24 @muhwan MODIFY
    // ctx.AbortWithStatusJSON(http.StatusUnauthorized, &defs.APIError{
    ctx.AbortWithStatusJSON(http.StatusForbidden, &defs.APIError{
        Status: "error",
        Error:  "authentication error",
    })

/mediamtx-1.15.6/internal/defs/path_access_request.go:
type PathAccessRequest struct {
	File             string // 2026.02.23 @muhwan	ADD

func (r *PathAccessRequest) ToAuthRequest() *auth.Request {    
	File             string // 2026.02.23 @muhwan	ADD


/mediamtx-1.15.6/internal/servers/rtsp/conn.go: 
	return &base.Response{
		// 2026.02.24 @muhwan MODIFY
		// StatusCode: base.StatusUnauthorized,
		StatusCode: base.StatusForbidden,
	}, err