.PHONY: build build_amd64 build_i386 install_build_dependencies

MODULE = github.com/xyaren/arcdps-log-uploader/cmd/arcdps-log-uploader

build: build_amd64 build_i386

build_amd64: install_build_dependencies
	GOOS=windows GOARCH=amd64 go generate $MODULE
	GOOS=windows GOARCH=amd64 go build -o ./out/arcdps-log-uploader_amd64.exe -ldflags="-H windowsgui" -i $MODULE

build_i386: install_build_dependencies
	GOOS=windows GOARCH=386 go generate $MODULE
	GOOS=windows GOARCH=386 go build -o ./out/arcdps-log-uploader_i386.exe -ldflags="-H windowsgui" -i $MODULE

install_build_dependencies:
	go get github.com/josephspurrier/goversioninfo/cmd/goversioninfo
