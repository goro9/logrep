package monidf

import (
	"flag"
	"fmt"
	"strings"
	"time"

	logex "github.com/goro9/logrep/pkg/logex"
)

func Monidf() {
	flag.Parse()
	var dir string
	if args := flag.Args(); len(args) < logex.ARG_NUM {
		fmt.Println("input required arguments")
	} else {
		dir = args[0]
		outPath := args[1]
		tStart, _ := time.Parse(time.RFC3339, "2020-11-20T00:00:00+09:00")
		tEnd, _ := time.Parse(time.RFC3339, "2020-12-31T00:00:00+09:00")
		le := logex.LogExplorer{
			Dir:                  dir,
			Target:               "queue is full",
			RowNum:               20,
			VersionConstraintStr: ">=1.2.4",
			FilterTimeStart:      tStart,
			FilterTimeEnd:        tEnd,
		}
		lers, _ := le.Logrep()

		err := logex.CreateFile(outPath, lers)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func VersionParser(l string) (bool, string) {
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

func Parser(l string) (logex.LogBuffer, error) {
	buf := strings.SplitN(l, " : ", 2)
	if len(buf) == 1 {
		return logex.LogBuffer{
			Log: l,
		}, nil
	}

	timeString := strings.Replace(buf[0], " ", "T", 1)
	timeString = strings.Replace(timeString, ",", ".", 1)
	// TODO: deal with location
	timeString = timeString + "+09:00"
	time, err := time.Parse(time.RFC3339Nano, timeString)
	if err != nil {
		fmt.Println(err)
		return logex.LogBuffer{}, err
	}

	log := buf[1]

	return logex.LogBuffer{
		Time: time,
		Log:  log,
	}, nil
}
