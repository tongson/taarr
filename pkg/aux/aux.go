package aux

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

func RunCmd(r RunArgs) (bool, string, string) {
	ret := true
	cmd := exec.Command(r.Exe, r.Args...)
	if len(r.Dir) > 0 {
		cmd.Dir = r.Dir
	}
	if r.Env != nil || len(r.Env) > 0 {
		cmd.Env = append(os.Environ(), r.Env...)
	}
	if r.Input != nil || len(r.Input) > 0 {
		cmd.Stdin = bytes.NewBuffer(r.Input)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		ret = false
	}
	outStr, errStr := stdout.String(), stderr.String()
	return ret, outStr, errStr
}

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

func InsertStr(a []string, b string, i int) []string {
	a = append(a, "")
	copy(a[i+1:], a[i:]) // number of elements copied ignored
	a[i] = b
	return a
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

func Pipestr(s string) string {
	str := strings.Replace(s, "\n", "\n | ", -1)
	str = " | \n | " + str
	return str + "\n | "
}

func StringToFile(filepath, s string) error {
	fo, err := os.Create(filepath)
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
