
package = ./cmd/arcdps-log-uploader
ldflags = "-H windowsgui -s -w"

.PHONY: build
build: build_amd64 build_i386

.PHONY: build_amd64
build_amd64: install_build_dependencies
	GOOS=windows GOARCH=amd64 go generate $(package)
	GOOS=windows GOARCH=amd64 go build -o ./out/arcdps-log-uploader_windows_amd64.exe -ldflags=$(ldflags) $(package)

.PHONY: build_i386
build_i386: install_build_dependencies
	GOOS=windows GOARCH=386 go generate $(package)
	GOOS=windows GOARCH=386 go build -o ./out/arcdps-log-uploader_windows_386.exe -ldflags=$(ldflags) $(package)

.PHONY: install_build_dependencies
install_build_dependencies:
	go get github.com/josephspurrier/goversioninfo/cmd/goversioninfo

.PHONY: generate
generate: install_build_dependencies
	go generate $(package)
