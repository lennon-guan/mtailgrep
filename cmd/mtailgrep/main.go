package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/nxadm/tail"
)

type tailLine struct {
	filename string
	line     *tail.Line
}

func main() {
	var keyword string
	flag.StringVar(&keyword, "keyword", "", "要过滤的关键词")
	flag.Parse()
	lc := make(chan tailLine)
	for _, filename := range flag.Args() {
		go startTail(filename, lc)
	}
	for l := range lc {
		if keyword == "" || strings.Contains(l.line.Text, keyword) {
			fmt.Printf("%s:%s\n", l.filename, l.line.Text)
		}
	}
}

func startTail(filename string, lc chan tailLine) {
	t, err := tail.TailFile(filename, tail.Config{
		Follow: true,
		ReOpen: true,
		Location: &tail.SeekInfo{
			Whence: io.SeekEnd,
			Offset: 0,
		},
	})
	if err != nil {
		panic(err)
	}
	for line := range t.Lines {
		lc <- tailLine{filename: filename, line: line}
	}
}
