module github.com/yhonda-ohishi-pub-dev/desktop-server

go 1.25.1

require (
	github.com/go-sql-driver/mysql v1.9.3
	github.com/improbable-eng/grpc-web v0.15.0
	github.com/microsoft/go-mssqldb v1.8.2
	github.com/yhonda-ohishi/db_service v1.9.1
	github.com/yhonda-ohishi/dtako_rows/v3 v3.4.2
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
)

require (
	github.com/yhonda-ohishi/dtako_events v1.6.1
	github.com/yhonda-ohishi/etc_data_processor v1.0.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	gorm.io/driver/sqlserver v1.6.1 // indirect
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/cenkalti/backoff/v4 v4.1.1 // indirect
	github.com/desertbit/timer v0.0.0-20180107155436-c41aec40b27f // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.3 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/joho/godotenv v1.5.1
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/rs/cors v1.7.0
	github.com/yhonda-ohishi-pub-dev/etc_meisai_scraper v0.0.31
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251022142026-3a174f9686a8 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251022142026-3a174f9686a8 // indirect
	gorm.io/driver/mysql v1.5.2 // indirect
	gorm.io/gorm v1.30.0 // indirect
	nhooyr.io/websocket v1.8.6 // indirect
)

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20240227224415-6ceb2ff114de
