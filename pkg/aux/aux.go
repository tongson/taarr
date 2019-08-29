package aux

import (
        "bytes"
        "os"
        "os/exec"
        "strings"
        "io/ioutil"
        "log"
)

type RunArgs struct {
        Exe   string
        Args  []string
        Env   []string
        Input []byte
}

func RunCmd(r RunArgs) (bool, string, string) {
	ret := true
	cmd := exec.Command(r.Exe, r.Args...)
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
	outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
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
			defer file.Close()
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
            defer file.Close()
            str, err := ioutil.ReadAll(file)
            if err != nil {
                    log.Panic(err)
            }
            return string(str)
        } else {
            return  ""
        }
}

func InsertStr(a []string, b string, i int) []string {
	a = append(a, "")
	copy(a[i+1:], a[i:])
	a[i] = b
	return a
}
