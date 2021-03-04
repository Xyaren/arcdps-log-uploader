build: build_rsrc
	GOOS=windows GOARCH=386 go build -o ./out/arc-link-sorter.exe -ldflags="-H windowsgui" -i ./cmd/arc-link-sorter

build_rsrc:
	GOOS=windows GOARCH=386 go run github.com/akavel/rsrc -manifest ./cmd/arc-link-sorter/arc-link-sorter.exe.manifest -o ./cmd/arc-link-sorter/rsrc.syso
