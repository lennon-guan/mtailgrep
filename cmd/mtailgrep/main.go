package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/nxadm/tail"
)

type tailLine struct {
	filename string
	line     *tail.Line
}

const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Gray    = "\033[37m"
	White   = "\033[97m"
)

var (
	filenameProcFunc = map[string]func(string) string{
		"none": func(string) string { return "" },
		"base": func(f string) string { return filepath.Base(f) },
		"full": func(f string) string { return f },
	}
)

func main() {
	var (
		keyword, fileStyle string
		colorName          bool
	)
	flag.StringVar(&keyword, "keyword", "", "要过滤的关键词")
	flag.StringVar(&fileStyle, "filestyle", "base", "文件名显示样式，none/base/full")
	flag.BoolVar(&colorName, "colorName", true, "是否用颜色显示文件名")
	flag.Parse()
	fpf := filenameProcFunc[fileStyle]
	if fpf == nil {
		panic("unsupported file style " + fileStyle)
	}
	outputFmt := "%s:%s\n"
	if colorName && fileStyle != "none" {
		outputFmt = "\033[32m%s\033[0m:%s\n"
	}
	lc := make(chan tailLine)
	for _, filename := range flag.Args() {
		go startTail(filename, lc)
	}
	if keyword == "" {
		for l := range lc {
			fmt.Printf(outputFmt, fpf(l.filename), l.line.Text)
		}
	} else {
		for l := range lc {
			if strings.Contains(l.line.Text, keyword) {
				fmt.Printf(outputFmt, fpf(l.filename), l.line.Text)
			}
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
