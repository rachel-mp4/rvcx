module xcvr-backend

go 1.22.1

require github.com/rachel-mp4/lrc/lrcd v0.0.0-20250410002721-ca6a18431212

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/rachel-mp4/lrc/lrc v0.0.0-20250408013928-75dc71a6060f // indirect
)

replace github.com/rachel-mp4/lrc/lrcd => ../../lrc/lrcd

replace github.com/rachel-mp4/lrc/lrc => ../../lrc/lrc
