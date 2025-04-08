module xcvr-backend

go 1.22.1

require github.com/rachel-mp4/lrc/lrcd v0.0.0-20250408005617-2d344a3d04f7

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/rachel-mp4/lrc/lrc v0.0.0-20250408005617-2d344a3d04f7 // indirect
)

replace github.com/rachel-mp4/lrc/lrcd => ../../lrc/lrcd

replace github.com/rachel-mp4/lrc/lrc => ../../lrc/lrc
