.PHONY: mtailgrep

mtailgrep:
	go build -o bin/mtailgrep cmd/mtailgrep/*.go
