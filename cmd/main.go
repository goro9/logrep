package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	flag.Parse()
	var dir string
	if args := flag.Args(); len(args) == 0 {
		fmt.Println("input target directory path")
	} else {
		dir = args[0]
		logrep(dirwalk(dir))
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

func logrep(paths []string) error {
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()

		reader := bufio.NewReader(f)

		for {
			lb, _, err := reader.ReadLine()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}

			l := string(lb)
			if strings.Contains(l, "W") {
				fmt.Println(p, reader.Size()-reader.Buffered(), l)
			}
		}
	}
	return nil
}
