module xcvr-backend

go 1.22.1

require github.com/rachel-mp4/lrc/lrcd v0.0.0-20250410194244-d1bffda40b72

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.4
	github.com/joho/godotenv v1.5.1
	github.com/rachel-mp4/lrc/lrc v0.0.0-20250408013928-75dc71a6060f // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)

replace github.com/rachel-mp4/lrc/lrcd => ../../lrc/lrcd

replace github.com/rachel-mp4/lrc/lrc => ../../lrc/lrc
