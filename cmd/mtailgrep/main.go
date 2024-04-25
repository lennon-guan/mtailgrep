package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lennon-guan/filterql"
	"github.com/nxadm/tail"
	"github.com/samber/lo"
)

type (
	tailLine struct {
		filename string
		line     *tail.Line
	}
	keywords []string
)

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

func main() {
	var (
		kwList, reList keywords
		whenceName     string
		fileStyle      string
		filter         string
		filterCond     filterql.BoolAst
		err            error
		colorName      bool
	)
	flag.Var(&kwList, "keyword", "要过滤的关键词")
	flag.Var(&reList, "re", "要过滤的正则表达式")
	flag.StringVar(&filter, "filter", "", "过滤消息用的filterql")
	flag.StringVar(&whenceName, "whence", "end", "开始tail的文件位置，start/current/end")
	flag.StringVar(&fileStyle, "filestyle", "base", "文件名显示样式，none/base/full")
	flag.BoolVar(&colorName, "colorName", true, "是否用颜色显示文件名")
	flag.Parse()
	fpf := lo.Switch[string, func(string) string](fileStyle).
		Case("none", func(string) string { return "" }).
		Case("base", func(f string) string { return filepath.Base(f) }).
		Case("full", func(f string) string { return f }).
		DefaultF(func() func(string) string {
			panic("unsupported file style " + fileStyle)
			return nil
		})
	whence := lo.Switch[string, int](whenceName).
		Case("start", io.SeekStart).
		Case("current", io.SeekCurrent).
		Case("end", io.SeekEnd).
		DefaultF(func() int {
			panic("unsupported whence " + whenceName)
			return -1
		})
	outputFmt := "%s:%s\n"
	if colorName && fileStyle != "none" {
		outputFmt = "\033[32m%s\033[0m:%s\n"
	}
	regexps := make([]*regexp.Regexp, len(reList))
	for i, re := range reList {
		regexps[i] = regexp.MustCompile(re)
	}
	if filter != "" {
		if filterCond, err = filterql.Parse(filter, &fqlConfig); err != nil {
			panic(err)
		}
	}
	lc := make(chan tailLine)
	for _, filename := range flag.Args() {
		go startTail(filename, whence, lc)
	}
	fqlCtx := filterql.NewContext(nil)
iterline:
	for l := range lc {
		for _, kw := range kwList {
			if !strings.Contains(l.line.Text, kw) {
				continue iterline
			}
		}
		for _, re := range regexps {
			if !re.MatchString(l.line.Text) {
				continue iterline
			}
		}
		if filterCond != nil {
			fqlCtx.Env = l.line.Text
			if matched, err := filterCond.IsTrue(fqlCtx); err != nil {
				panic(err)
			} else if !matched {
				continue iterline
			}
		}
		fmt.Printf(outputFmt, fpf(l.filename), l.line.Text)
	}
}

func startTail(filename string, whence int, lc chan tailLine) {
	t, err := tail.TailFile(filename, tail.Config{
		Follow: true,
		ReOpen: true,
		Location: &tail.SeekInfo{
			Whence: whence,
			Offset: 0,
		},
		Logger: tail.DiscardingLogger,
	})
	if err != nil {
		panic(err)
	}
	for line := range t.Lines {
		lc <- tailLine{filename: filename, line: line}
	}
}

func fqlMatch(env any, p string) (any, error) {
	re, ok := fqlReMap[p]
	if !ok {
		var err error
		re, err = regexp.Compile(p)
		if err != nil {
			return false, err
		}
		fqlReMap[p] = re
	}
	if re == nil {
		return false, nil
	}
	return re.MatchString(env.(string)), nil
}

var (
	fqlReMap  = map[string]*regexp.Regexp{}
	fqlConfig = filterql.ParseConfig{
		StrMethods: map[string]func(any, string) (any, error){
			"keyword": func(env any, kw string) (any, error) {
				return strings.Contains(env.(string), kw), nil
			},
			"ikeyword": func(env any, kw string) (any, error) {
				return strings.Contains(strings.ToLower(env.(string)), strings.ToLower(kw)), nil
			},
			"match": fqlMatch,
			"imatch": func(env any, p string) (any, error) {
				return fqlMatch(env, "(?i)"+p)
			},
		},
		IntMethods: map[string]func(any, int) (any, error){},
	}
)
