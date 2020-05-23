package lib

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

type RunArgs struct {
	Exe   string
	Args  []string
	Dir   string
	Env   []string
	Input []byte
}

type panicT struct {
	msg  string
	code int
}

// Interface to execute the given `RunArgs` through `exec.Command`.
// The first return value is a boolean, true indicates success, false otherwise.
// Second value is the standard output of the command.
// Third value is the standard error of the command.
func (a RunArgs) Run() (bool, string, string) {
	var r bool = true
	cmd := exec.Command(a.Exe, a.Args...)
	if len(a.Dir) > 0 {
		cmd.Dir = a.Dir
	}
	if a.Env != nil || len(a.Env) > 0 {
		cmd.Env = append(os.Environ(), a.Env...)
	}
	if a.Input != nil || len(a.Input) > 0 {
		cmd.Stdin = bytes.NewBuffer(a.Input)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		r = false
	}
	outStr, errStr := stdout.String(), stderr.String()
	return r, outStr, errStr
}

// Returns a function for simple directory or file check.
// StatPath("directory") for directories.
// StatPath() or StatPath("whatever") for files.
// The function returns boolean `true` on successfully check, `false` otherwise.
func StatPath(f string) func(string) bool {
	switch f {
	case "directory":
		return func(i string) bool {
			if fi, err := os.Stat(i); err == nil {
				if fi.IsDir() {
					return true
				}
			}
			return false
		}
	default:
		return func(i string) bool {
			info, err := os.Stat(i)
			if os.IsNotExist(err) {
				return false
			}
			return !info.IsDir()
		}
	}
}

// Returns a function for walking a path for files.
// Files are read and then contents are written to a strings.Builder pointer.
func PathWalker(sh *strings.Builder) func(string, os.FileInfo, error) error {
	isFile := StatPath("file")
	return func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if isFile(path) {
			file, err := os.Open(path)
			if err != nil {
				log.Panic(err)
			}
			defer func() {
				err := file.Close()
				if err != nil {
					log.Panic(err)
				}
			}()
			str, err := ioutil.ReadAll(file)
			if err != nil {
				log.Panic(err)
			}
			sh.WriteString(string(str)) // length of string and nil err ignored
		}
		return nil
	}
}

// Reads a file `path` then returns the contents as a string.
// Always returns a string value.
// An empty string "" is returned for nonexistent or unreadable files.
func FileRead(path string) string {
	isFile := StatPath("file")
	if isFile(path) {
		file, err := os.Open(path)
		if err != nil {
			log.Panic(err)
		}
		defer func() {
			err := file.Close()
			if err != nil {
				log.Panic(err)
			}
		}()
		str, err := ioutil.ReadAll(file)
		if err != nil {
			log.Panic(err)
		}
		return string(str)
	} else {
		return ""
	}
}

// Insert string argument #2 into index `i` of first argument `a`.
func InsertStr(a []string, b string, i int) []string {
	a = append(a, "")
	copy(a[i+1:], a[i:]) // number of elements copied ignored
	a[i] = b
	return a
}

// Prefix string `s` with pipes "|".
// Used to "prettify" command line output.
// Returns new string.
func PipeStr(s string) string {
	str := strings.Replace(s, "\n", "\n | ", -1)
	str = " | \n | " + str
	return str + "\n | "
}

// Writes the string `s` to the file `path`.
// It returns any error encountered, nil otherwise.
func StringToFile(path string, s string) error {
	fo, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = fo.Close()
	}()
	_, err = io.Copy(fo, strings.NewReader(s))
	if err != nil {
		return err
	}
	return nil
}

func RecoverPanic() {
	if rec := recover(); rec != nil {
		err := rec.(panicT)
		fmt.Fprintln(os.Stderr, err.msg)
		os.Exit(err.code)
	}
}

func Assert(e error, s string) {
	if e != nil {
		strerr := strings.Replace(e.Error(), "\n", "\n | ", -1)
		panic(panicT{msg: fmt.Sprintf("assertion failed: %s\n | \n | %s\n | ", s, strerr), code: 255})
	}
}

func Bug(s string) {
	panic(panicT{msg: fmt.Sprintf("bug: %s", s), code: 255})
}

func Panic(s string) {
	panic(panicT{msg: fmt.Sprintf("fatal error: %s", s), code: 1})
}

func Panicf(f string, a ...interface{}) {
	panic(panicT{msg: fmt.Sprintf("fatal error: "+f, a...), code: 1})
}
