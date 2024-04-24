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

type keywords []string

func (kws *keywords) String() string {
	return "[" + strings.Join([]string(*kws), ", ") + "]"
}

func (kws *keywords) Set(v string) error {
	*kws = append(*kws, v)
	return nil
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
		kws       keywords
		fileStyle string
		colorName bool
	)
	flag.Var(&kws, "keyword", "要过滤的关键词")
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
	if len(kws) == 0 {
		for l := range lc {
			fmt.Printf(outputFmt, fpf(l.filename), l.line.Text)
		}
	} else {
	iterline:
		for l := range lc {
			for _, kw := range kws {
				if !strings.Contains(l.line.Text, kw) {
					continue iterline
				}
			}
			fmt.Printf(outputFmt, fpf(l.filename), l.line.Text)
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
