module xcvr-backend

go 1.22.1

require github.com/rachel-mp4/lrc/lrcd v0.0.0-20250410194244-d1bffda40b72

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/rachel-mp4/lrc/lrc v0.0.0-20250408013928-75dc71a6060f // indirect
)

replace github.com/rachel-mp4/lrc/lrcd => ../../lrc/lrcd

replace github.com/rachel-mp4/lrc/lrc => ../../lrc/lrc
