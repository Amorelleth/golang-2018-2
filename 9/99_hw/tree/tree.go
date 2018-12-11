package main

import (
	"fmt"
	"io"
	"os"
	"sort"
)

type FI struct {
	FI []os.FileInfo
}

func (fi *FI) Len() int {
	return len(fi.FI)
}

func (fi *FI) Swap(i, j int) {
	fi.FI[i], fi.FI[j] = fi.FI[j], fi.FI[i]
}

func (fi *FI) Less(i, j int) bool {
	return fi.FI[i].Name() < fi.FI[j].Name()
}

func printDir(out io.Writer, path string, print bool, prefix string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	res, err := file.Readdir(0)
	if err != nil {
		return err
	}

	fi := FI{res}
	fileInfo := fi.FI
	sort.Sort(&fi)

	for i, r := range fileInfo {
		IsDir := r.IsDir()
		IsLast := i == len(fileInfo)-1

		if !(print || IsDir) {
			continue
		}

		IsLastDir := true
		for j := i + 1; j < len(fileInfo); j++ {
			if fileInfo[j].IsDir() {
				IsLastDir = false
				break
			}
		}

		if !print && IsLastDir || IsLast {
			fmt.Fprintf(out, "%s└───%s", prefix, r.Name())
		} else {
			fmt.Fprintf(out, "%s├───%s", prefix, r.Name())
		}

		if IsDir {
			fmt.Fprint(out, "\n")
			p := "\t"

			if !print {
				if !IsLastDir {
					p = "│\t"
				}
			} else if !IsLast {
				p = "│\t"
			}
			printDir(out, path+string(os.PathSeparator)+r.Name(), print, prefix+p)
		} else {
			size := fmt.Sprintf("%vb", r.Size())
			if size == "0b" {
				size = "empty"
			}
			fmt.Fprintf(out, " (%s)\n", size)
		}
	}
	return nil
}

func dirTree(out io.Writer, path string, print bool) error {
	err := printDir(out, path, print, "")
	if err != nil {
		return err
	}
	return nil
}
