.PHONY: proto clean build build-gui run release install-tools update-frontend

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/database.proto

clean:
	rm -rf proto/*.pb.go
	rm -rf desktop-server.exe
	rm -rf dist/
	rm -rf frontend/dist/

build: proto
	go mod tidy
	go build -o desktop-server.exe .

build-gui: proto
	go mod tidy
	go build -ldflags="-H windowsgui" -o desktop-server.exe .

run: build
	./desktop-server.exe

release: build-gui
	@echo "Creating release build..."
	@mkdir -p dist
	@cp desktop-server.exe dist/desktop-server.exe
	@echo "Release build created: dist/desktop-server.exe"
	@echo "Upload this to GitHub Releases"

install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

update-frontend:
	@echo "Downloading latest frontend..."
	go run . -update
