package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	lib "github.com/tongson/gl"
)

const versionNumber = "0.6.0"
const codeName = "\"Jawless Keg\""
const (
	libHeader = `
unset IFS
set -o errexit -o nounset -o noglob
export PATH=/bin:/sbin:/usr/bin:/usr/sbin
export LC_ALL=C
`
)
const run = "script"

type logWriter struct {
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(time.Now().Format(time.RFC1123Z) + " [debug] " + string(bytes))
}

func output(o string, h string, c string) (string, string) {
	rh := ""
	rb := ""
	if o != "" {
		rh = fmt.Sprintf(" %s%s\n", h, c)
		rb = fmt.Sprintf("%s\n", lib.PipeStr(h, o))
	}
	return rh, rb
}

func main() {
	var verbose bool = false
	var failed bool = false
	runtime.MemProfileRate = 0
	defer lib.RecoverPanic()
	log.SetFlags(0)
	call := os.Args[0]
	if len(call) < 3 || call[len(call)-2:] == "rr" {
		log.SetOutput(io.Discard)
	} else if call[len(call)-3:] == "rrv" {
		verbose = true
		log.SetOutput(new(logWriter))
	} else {
		lib.Bug("Unsupported executable name.")
	}
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		verbose = true
		log.SetOutput(new(logWriter))
	}
	log.Printf("rr %s %s", versionNumber, codeName)

	isDir := lib.StatPath("directory")
	isFile := lib.StatPath("file")
	var sh strings.Builder

	var offset int
	var hostname string
	if len(os.Args) < 2 {
		lib.Panic("Missing arguments.")
	}
	if strings.Contains(os.Args[1], "/") || strings.Contains(os.Args[1], ":") {
		offset = 1
		hostname = "local"
	} else {
		offset = 2
		hostname = os.Args[1]
	}
	if len(os.Args) < offset+1 {
		lib.Panic("`namespace:script` not specified.")
	}
	s := strings.Split(os.Args[offset], "/")
	if len(s) < 2 {
		s = strings.Split(os.Args[offset], ":")
	}
	if len(s) < 2 {
		lib.Panic("`namespace:script` not specified.")
	}
	namespace, script := s[0], s[1]
	if !isDir(namespace) {
		lib.Panicf("`%s`(namespace) is not a directory.", namespace)
	}
	if !isDir(fmt.Sprintf("%s/%s", namespace, script)) {
		lib.Panicf("`%s/%s` is not a diretory.", namespace, script)
	}
	if !isFile(fmt.Sprintf("%s/%s/%s", namespace, script, run)) {
		lib.Panicf("`%s/%s/%s` actual script not found.", namespace, script, run)
	}
	var arguments []string
	if len(s) > 2 {
		arguments = []string{}
		arguments = append(arguments, s[2])
		arguments = append(arguments, os.Args[offset+1:]...)
	} else {
		arguments = os.Args[offset+1:]
	}
	fnwalk := lib.PathWalker(&sh)
	if isDir(".lib") {
		lib.Assert(filepath.Walk(".lib", fnwalk), "filepath.Walk(\".lib\")")
	}

	if isDir(namespace + "/.lib") {
		lib.Assert(filepath.Walk(namespace+"/.lib", fnwalk), "filepath.Walk(namespace+\".lib\")")
	}
	if isDir(namespace + "/" + script + "/.lib") {
		lib.Assert(filepath.Walk(namespace+"/"+script+"/.lib", fnwalk), "filepath.Walk(namespace+\".lib\")")
	}

	//Pass environment variables with `rr` prefix
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "rr") {
			sh.WriteString("export " + strings.TrimPrefix(e, "rr") + "\n")
		}
	}

	arguments = lib.InsertStr(arguments, "set --", 0)
	sh.WriteString(strings.Join(arguments, " "))
	sh.WriteString("\n" + lib.FileRead(namespace+"/"+script+"/"+run))
	modscript := sh.String()
	//print debugging -- fmt.Println(modscript)
	const STDOUT = " >>>  STDOUT  >>>"
	const STDERR = " >>>  STDERR  >>>"
	log.Printf("Running %s:%s via %s", namespace, script, hostname)
	if hostname == "local" || hostname == "localhost" {
		untar := `
                LC_ALL=C
                set -o errexit -o nounset -o noglob
                unset IFS
                PATH=/bin:/usr/bin
                tar -C %s -cpf - . | tar -C / -xpf -
                `
		for _, d := range []string{
			".files",
			".files-local",
			".files-localhost",
			namespace + "/.files",
			namespace + "/.files-local",
			namespace + "/.files-localhost",
			namespace + "/" + script + "/.files",
			namespace + "/" + script + "/.files-local",
			namespace + "/" + script + "/.files-localhost",
		} {
			if isDir(d) {
				rargs := lib.RunArgs{Exe: "sh", Args: []string{"-c", fmt.Sprintf(untar, d)}}
				ret, stdout, stderr, _ := rargs.Run()
				if !ret {
					failed = true
					if !verbose {
						lib.Panicf("Error copying files::\nstdout::\n%s\nstderr::\n%s\n", stdout, stderr)
					} else {
						ho, bo := output(stdout, hostname, STDOUT)
						he, be := output(stderr, hostname, STDERR)
						log.Printf("Failure copying files!\n%s%s%s%s", ho, bo, he, be)
					}
				}
			}
		}
		rargs := lib.RunArgs{Exe: "sh", Args: []string{"-c", modscript}}
		ret, stdout, stderr, _ := rargs.Run()
		if !ret {
			failed = true
			if !verbose {
				lib.Panicf("\nstdout::\n%s\nstderr::\n%s\n", stdout, stderr)
			} else {
				ho, bo := output(stdout, hostname, STDOUT)
				he, be := output(stderr, hostname, STDERR)
				log.Printf("Failure running script!\n%s%s%s%s", ho, bo, he, be)
			}
		} else {
			ho, bo := output(stdout, hostname, STDOUT)
			he, be := output(stderr, hostname, STDERR)
			if stdout != "" && stderr != "" {
				log.Printf("Output:\n%s%s%s%s", ho, bo, he, be)
			}
		}
	} else {
		rh := strings.Split(hostname, "@")
		var realhost string
		if len(rh) == 1 {
			realhost = hostname
		} else {
			realhost = rh[1]
		}
		sshenv := []string{"LC_ALL=C"}
		ssha := lib.RunArgs{Exe: "ssh", Args: []string{"-T", "-a", "-x", "-C", hostname, "uname -n"}, Env: sshenv}
		ret, stdout, _, _ := ssha.Run()
		if ret {
			sshhost := strings.Split(stdout, "\n")
			if realhost != sshhost[0] {
				lib.Panicf("Hostname %s does not match remote host.", realhost)
			} else {
				log.Printf("Remote host is %s\n", sshhost[0])
			}
		} else {
			if !verbose {
				lib.Panicf("%s does not exist or unreachable.", realhost)
			} else {
				log.Printf("%s does not exist or unreachable.", realhost)
			}
		}
		for _, d := range []string{
			".files",
			".files-" + realhost,
			namespace + "/.files",
			namespace + "/.files-" + realhost,
			namespace + "/" + script + "/.files",
			namespace + "/" + script + "/.files-" + realhost,
		} {
			if isDir(d) {
				log.Printf("Copying %s to %s...", d, realhost)
				tmpfile, err := os.CreateTemp(os.TempDir(), "_rr")
				if err != nil {
					lib.Panic("Cannot create temporary file.")
				}
				defer os.Remove(tmpfile.Name())
				sftpc := []byte(fmt.Sprintf("lcd %s\ncd /\nput -fRp .\n bye\n", d))
				if _, err = tmpfile.Write(sftpc); err != nil {
					lib.Panic("Failed to write to temporary file.")
				}
				tmpfile.Close()
				sftpa := lib.RunArgs{Exe: "sftp", Args: []string{"-C", "-b", tmpfile.Name(), hostname}, Env: sshenv}
				ret, _, _, _ := sftpa.Run()
				if !ret {
					lib.Panic("Running sftp failed.")
				}
				os.Remove(tmpfile.Name())
			}
		}
		log.Println("Running script...")
		sshb := lib.RunArgs{Exe: "ssh", Args: []string{"-T", "-a", "-x", "-C", hostname}, Env: sshenv,
			Stdin: []byte(modscript)}
		ret, stdout, stderr, _ := sshb.Run()
		ho, bo := output(stdout, hostname, STDOUT)
		he, be := output(stderr, hostname, STDERR)
		if !ret {
			failed = true
			if !verbose {
				lib.Panicf("\nstdout::\n%s\nstderr::\n%s\n", stdout, stderr)
			} else {
				if stdout != "" && stderr != "" {
					log.Printf("Output:\n%s%s%s%s", ho, bo, he, be)
				}
			}
		} else {
			if stdout != "" && stderr != "" {
				log.Printf("Output:\n%s%s%s%s", ho, bo, he, be)
			}
		}
	}
	if !failed {
		log.Println("All OK.")
	} else {
		log.Println("Something went wrong.")
	}
}
