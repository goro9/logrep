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

	"github.com/hashicorp/go-version"
)

type LogBufferFull struct {
	Row int
	LogBuffer
}

func (lb LogBufferFull) String() string {
	return fmt.Sprintf("\t%v: time=%v, log=%v", lb.Row, lb.Time.Format(time.RFC3339Nano), lb.Log)
}

type LogBuffer struct {
	Time time.Time
	Log  string
}

type LogExplorerResult struct {
	Path string
	Ver  *version.Version
	Logs []LogBufferFull
}

func (ler LogExplorerResult) String() string {
	bufs := []string{fmt.Sprintf("ver=%v, path=%v", ler.Ver, ler.Path)}
	for _, log := range ler.Logs {
		bufs = append(bufs, fmt.Sprintf("\n%v", log))
	}
	return strings.Join(bufs, "")
}

type LogExplorer struct {
	Dir    string
	Target string
	RowNum int
}

type context struct {
	path    string
	target  string
	ver     *version.Version
	ringBuf *ring.Ring
	result  []LogExplorerResult
}

const (
	ARG_NUM = 2
)

func main() {
	flag.Parse()
	var dir string
	if args := flag.Args(); len(args) < ARG_NUM {
		fmt.Println("input required arguments")
	} else {
		dir = args[0]
		outPath := args[1]
		le := LogExplorer{
			Dir:    dir,
			Target: "W",
			RowNum: 10,
		}
		lers, _ := le.logrep()

		err := createFile(outPath, lers)
		if err != nil {
			fmt.Println(err)
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
		// TODO: catch up runnning software version
		ringBuf: ring.New(le.RowNum),
		target:  le.Target,
	}

	for _, path := range dirwalk(le.Dir) {
		// TODO: filter by file timestamp
		finfo, err := os.Stat(path)
		fmt.Println(finfo.ModTime())

		ctx.path = path
		err = searchFile(&ctx)
		if err != nil {
			return nil, err
		}
	}
	return &ctx.result, nil
}

func createFile(path string, lers *[]LogExplorerResult) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for i, ler := range *lers {
		_, err = f.WriteString(fmt.Sprintf("%v: %v\n", i, ler))
		if err != nil {
			return err
		}
	}
	return nil
}

func searchFile(ctx *context) error {
	f, err := os.Open(ctx.path)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	row := 0
	for {
		row++
		lb, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		logBuf, err := newLogBuffer(string(lb))
		if err != nil {
			continue
		}
		ctx.ringBuf.Value = LogBufferFull{
			Row:       row,
			LogBuffer: logBuf,
		}

		if isVer, ver := getVersion(string(lb)); isVer {
			ctx.ver, _ = version.NewVersion(ver)
			if err != nil {
				fmt.Printf("version log pattern found but invalid version format: %v", err)
			}
		}

		if strings.Contains(ctx.ringBuf.Value.(LogBufferFull).Log, ctx.target) {
			var ler LogExplorerResult
			ler.Path = ctx.path
			ler.Ver = ctx.ver
			ctx.ringBuf.Do((func(v interface{}) {
				if v == nil {
					return
				}
				ler.Logs = append(ler.Logs, v.(LogBufferFull))
			}))
			ctx.result = append(ctx.result, ler)
		}
		ctx.ringBuf = ctx.ringBuf.Next()
	}
	return nil
}

func getVersion(l string) (bool, string) {
	containsString := "cpu_start: App version:"
	hit := strings.Contains(l, containsString)
	version := "unknown"
	if hit {
		buf := strings.Split(l, " ")
		version = buf[len(buf)-1]
		// remove "v"
		version = version[1:]
		// remove ANSI escape sequence
		version = strings.Replace(version, "\033[0m", "", 1)
	}
	return hit, version
}

func newLogBuffer(l string) (LogBuffer, error) {
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
	time, err := time.Parse(time.RFC3339Nano, timeString)
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
