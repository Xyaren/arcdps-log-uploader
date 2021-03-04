build: download_modules build_rsrc build_amd64 build_386

download_modules:
	go get ./...

build_amd64:
	GOOS=windows GOARCH=amd64 go build -o ./out/arc-link-sorter_amd64.exe -ldflags="-H windowsgui" -i ./cmd/arc-link-sorter

build_386:
	GOOS=windows GOARCH=386 go build -o ./out/arc-link-sorter_i386.exe -ldflags="-H windowsgui" -i ./cmd/arc-link-sorter

build_rsrc:
	go run github.com/akavel/rsrc -manifest ./cmd/arc-link-sorter/arc-link-sorter.exe.manifest -o ./cmd/arc-link-sorter/rsrc.syso
