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

type logBuffer struct {
	Time time.Time
	Log  string
}

type LogExplorerResult struct {
	Version string
	Logs    []logBuffer
}

type LogExplorer struct {
	Paths  []string
	Target string
	RowNum int
}

func main() {
	flag.Parse()
	var dir string
	if args := flag.Args(); len(args) == 0 {
		fmt.Println("input target directory path")
	} else {
		dir = args[0]
		le := LogExplorer{
			dirwalk(dir),
			"W",
			5,
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
	var lers []LogExplorerResult
	// TODO: catch up runnning software version
	version := "unknown"
	rbLogs := ring.New(le.RowNum)
	for _, p := range le.Paths {
		f, err := os.Open(p)
		if err != nil {
			return nil, err
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
				return nil, err
			}

			rbLogs.Value = logBuffer{
				time.Now(), // TODO: get time from log
				string(lb), // TODO: get only text part of log
			}
			if strings.Contains(rbLogs.Value.(logBuffer).Log, "W") {
				var ler LogExplorerResult
				ler.Version = version
				rbLogs.Do((func(v interface{}) {
					if v == nil {
						return
					}
					ler.Logs = append(ler.Logs, v.(logBuffer))
				}))
				lers = append(lers, ler)
			}
			rbLogs = rbLogs.Next()
		}
	}
	return &lers, nil
}
