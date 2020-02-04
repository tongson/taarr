package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aux"
)

const versionNumber = "v0.0.1"
const codeName = "\"Tubeless Pope\""
const (
	libHeader = `#!/usr/bin/env bash
unset IFS
set -o errexit -o nounset -o pipefail -o errtrace
export PATH=/bin:/sbin:/usr/bin:/usr/sbin
export LC_ALL=C
`
)

type logWriter struct {
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(time.Now().UTC().Format("2006-01-02T15:04:05Z07:00") + " [debug] " + string(bytes))
}

func main() {
	defer aux.RecoverPanic()
	log.SetFlags(0)
	call := os.Args[0]
	if len(call) < 3 || call[len(call)-2:] == "rr" {
		log.SetOutput(ioutil.Discard)
	} else if call[len(call)-3:] == "rrv" {
		log.SetOutput(new(logWriter))
	} else {
		aux.Bug("unhandled os.Args[0] length or value.")
	}
	log.Printf("rr %s %s", versionNumber, codeName)

	isDir := aux.StatPath("directory")
	isFile := aux.StatPath("file")
	var sh strings.Builder

	if len(os.Args) < 2 {
		aux.Panic("Missing arguments. Exiting.")
	}
	if len(os.Args) < 3 {
		aux.Panic("`namespace:script` not specified. Exiting.")
	}
	hostname := os.Args[1]
	s := strings.Split(os.Args[2], "/")
	if len(s) < 2 {
		s = strings.Split(os.Args[2], ":")
	}
	if len(s) < 2 {
		aux.Panic("`namespace:script` not specified. Exiting.")
	}
	namespace, script := s[0], s[1]
	if !isDir(namespace) {
		aux.Panicf("`%s`(namespace) is not a directory. Exiting.", namespace)
	}
	if !isFile(fmt.Sprintf("%s/%s", namespace, script)) {
		aux.Panicf("`%s`(script) is not a file. Exiting.", script)
	}
	arguments := os.Args[3:]

	fnwalk := aux.PathWalker(&sh)
	if !isDir(".lib") {
		_ = os.MkdirAll(".lib", os.ModePerm)
		aux.Assert(aux.StringToFile(".lib/000-header.sh", libHeader), "Writing .lib/000-header.sh")
	}
	aux.Assert(filepath.Walk(".lib", fnwalk), "filepath.Walk(\".lib\")")

	if isDir(namespace + "/.lib") {
		aux.Assert(filepath.Walk(namespace+"/.lib", fnwalk), "filepath.Walk(namespace+\".lib\")")
	}
	arguments = aux.InsertStr(arguments, "set --", 0)
	sh.WriteString(strings.Join(arguments, " "))
	sh.WriteString("\n" + aux.FileRead(namespace+"/"+script))
	modscript := sh.String()
	//print debugging -- fmt.Println(modscript)
	log.Printf("Running %s:%s over %s", namespace, script, hostname)
	if hostname == "local" || hostname == "localhost" {
		untar := `
                LC_ALL=C
                set -o pipefail -o nounset -o errexit -o noglob
                unset IFS
                PATH=/bin:/usr/bin
                tar -C %s -cpf - . | tar -C / -xpf -
                `
		for _, d := range []string{".files", ".files-local", ".files-localhost", namespace + "/.files", namespace + "/.files-local", namespace + "/.files-localhost"} {
			if isDir(d) {
				rargs := aux.RunArgs{Exe: "bash", Args: []string{"-c", fmt.Sprintf(untar, d)}}
				ret, stdout, stderr := rargs.Run()
				if !ret {
					aux.Panicf("Failure copying files!\n  -- STDOUT --\n%s\n  -- STDERR --\n%s\n", aux.Pipestr(stdout), aux.Pipestr(stderr))
				}
			}
		}
		rargs := aux.RunArgs{Exe: "bash", Args: []string{"-c", modscript}}
		ret, stdout, stderr := rargs.Run()
		if !ret {
			aux.Panicf("Failure running script!\n  -- STDOUT --\n%s\n  -- STDERR --\n%s", aux.Pipestr(stdout), aux.Pipestr(stderr))
		} else {
			log.Printf("Output:\n  -- STDOUT --\n%s\n  -- STDERR --\n%s\n", aux.Pipestr(stdout), aux.Pipestr(stderr))
		}
	} else {
		sshenv := []string{"LC_ALL=C"}
		ssha := aux.RunArgs{Exe: "ssh", Args: []string{"-T", "-a", "-P", "-x", "-C", hostname, "uname -n"}, Env: sshenv}
		ret, stdout, _ := ssha.Run()
		if ret {
			sshhost := strings.Split(stdout, "\n")
			if hostname != sshhost[0] {
				aux.Panicf("hostname %s does not match remote host. Exiting.", hostname)
			} else {
				log.Printf("Remote host is %s\n", sshhost[0])
			}
		} else {
			aux.Panicf("%s does not exist or unreachable. Exiting.", hostname)
		}
		for _, d := range []string{".files", ".files-" + hostname, namespace + "/.files", namespace + "/.files-" + hostname} {
			if isDir(d) {
				log.Printf("Copying %s to %s...", d, hostname)
				sftpc := []byte(fmt.Sprintf("lcd %s\ncd /\nput -rP .\n bye\n", d))
				sftpa := aux.RunArgs{Exe: "sftp", Args: []string{"-C", "-b", "/dev/fd/0", hostname}, Env: sshenv, Input: sftpc}
				ret, _, _ := sftpa.Run()
				if !ret {
					aux.Panic("Running sftp failed. Exiting.")
				}
			}
		}
		log.Println("Running script...")
		sshb := aux.RunArgs{Exe: "ssh", Args: []string{"-T", "-a", "-P", "-x", "-C", hostname}, Env: sshenv, Input: []byte(modscript)}
		ret, stdout, stderr := sshb.Run()
		if !ret {
			aux.Panicf("Failure running script!\n  -- STDOUT --\n%s\n  -- STDERR --\n%s\n", aux.Pipestr(stdout), aux.Pipestr(stderr))
		} else {
			log.Printf("Output:\n  -- STDOUT --\n%s\n  -- STDERR --\n%s\n", aux.Pipestr(stdout), aux.Pipestr(stderr))

		}
	}
	log.Println("All OK.")
}
