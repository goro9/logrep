package main

import (
	"flag"
	"fmt"
	"time"

	logex "github.com/goro9/logrep/pkg/logex"
)

func main() {
	flag.Parse()
	var dir string
	if args := flag.Args(); len(args) < logex.ARG_NUM {
		fmt.Println("input required arguments")
	} else {
		dir = args[0]
		outPath := args[1]
		tStart, _ := time.Parse(time.RFC3339, "2020-08-20T00:00:00+09:00")
		tEnd, _ := time.Parse(time.RFC3339, "2020-08-21T00:00:00+09:00")
		le := logex.LogExplorer{
			Dir:                  dir,
			Target:               "W",
			RowNum:               10,
			VersionConstraintStr: ">=0",
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
