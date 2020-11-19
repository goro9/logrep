package main

import (
	"bufio"
	"container/ring"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type LogBuffer struct {
	Time time.Time
	Log  string
}

type LogExplorerResult struct {
	Version string
	Logs    []LogBuffer
}

type LogExplorer struct {
	Dir    string
	Target string
	RowNum int
}

type context struct {
	paths      []string
	path       string
	version    string
	ringBuf    *ring.Ring
	result     []LogExplorerResult
	timeLayout string
}

func main() {
	flag.Parse()
	var dir string
	if args := flag.Args(); len(args) == 0 {
		fmt.Println("input target directory path")
	} else {
		dir = args[0]
		le := LogExplorer{
			Dir:    dir,
			Target: "W",
			RowNum: 5,
		}
		lers, _ := le.logrep()
		fmt.Println("")
		for i, v := range *lers {
			fmt.Printf("%v: version=%v, logs=\n", i, v.Version)
			for _, log := range v.Logs {
				fmt.Printf("\ttime=%v, log=%v\n", log.Time, log.Log)
			}
		}
	}
}

func dirwalk(dir string) []string {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	var paths []string
	for _, file := range files {
		if file.IsDir() {
			paths = append(paths, dirwalk(filepath.Join(dir, file.Name()))...)
			continue
		}
		paths = append(paths, filepath.Join(dir, file.Name()))
	}

	sort.Strings(paths)
	return paths
}

func (le *LogExplorer) logrep() (*[]LogExplorerResult, error) {
	ctx := context{
		paths:      dirwalk(le.Dir),
		version:    "unknown", // TODO: catch up runnning software version
		ringBuf:    ring.New(le.RowNum),
		timeLayout: time.RFC3339Nano,
	}

	for _, path := range dirwalk(le.Dir) {
		ctx.path = path
		err := searchFile(&ctx)
		if err != nil {
			return nil, err
		}
	}
	return &ctx.result, nil
}

func searchFile(ctx *context) error {
	f, err := os.Open(ctx.path)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	lNum := 0
	for {
		lNum++
		lb, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		ctx.ringBuf.Value, err = newLogBuffer(ctx.timeLayout, string(lb))
		if err != nil {
			continue
		}

		if strings.Contains(ctx.ringBuf.Value.(LogBuffer).Log, "W") {
			var ler LogExplorerResult
			ler.Version = ctx.version
			ctx.ringBuf.Do((func(v interface{}) {
				if v == nil {
					return
				}
				ler.Logs = append(ler.Logs, v.(LogBuffer))
			}))
			ctx.result = append(ctx.result, ler)
		}
		ctx.ringBuf = ctx.ringBuf.Next()
	}
	return nil
}

func newLogBuffer(timeLayout string, l string) (LogBuffer, error) {
	buf := strings.SplitN(l, " : ", 2)
	if len(buf) == 1 {
		return LogBuffer{
			Log: l,
		}, nil
	}

	// TODO: deal with location
	timeString := strings.Replace(buf[0], " ", "T", 1)
	timeString = strings.Replace(timeString, ",", ".", 1)
	timeString = timeString + "Z"
	time, err := time.Parse(timeLayout, timeString)
	if err != nil {
		fmt.Println(err)
		return LogBuffer{}, err
	}

	log := buf[1]

	return LogBuffer{
		Time: time,
		Log:  log,
	}, nil
}
