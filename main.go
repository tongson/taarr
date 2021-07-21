package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	zerolog "github.com/rs/zerolog"
	lib "github.com/tongson/gl"
	spin "github.com/tongson/rr/external/go-spin"
)

var start = time.Now()

const versionNumber = "0.10.0"
const codeName = "\"Kilowatt Triceps\""
const run = "script"

type logWriter struct {
}

func showSpinnerWhile(s int) func() {
	spinner := spin.New()
	switch s {
	case 0:
		spinner.Set(spin.Spin24)
	default:
		spinner.Set(spin.Spin26)
	}
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
			default:
				fmt.Fprintf(os.Stderr, "%s\r", spinner.Next())
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
	return func() {
		done <- true
		fmt.Fprintf(os.Stderr, "\033[%dD", 1)
	}
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(time.Now().Format(time.RFC1123Z) + " " + string(bytes))
}

func output(o string, h string, c string) (string, string, string) {
	footer := " └──"
	rh := ""
	rb := ""
	rf := ""
	if o != "" {
		rh = fmt.Sprintf(" %s%s\n", h, c)
		rb = fmt.Sprintf("%s\n", lib.PipeStr(h, "│", o))
		rf = fmt.Sprintf(" %s%s\n", h, footer)
	}
	return rh, rb, rf
}

func main() {
	var errLog zerolog.Logger
	var verbose bool = false
	var failed bool = false
	var dump bool = false
	runtime.MemProfileRate = 0
	defer lib.RecoverPanic()
	log.SetFlags(0)
	call := os.Args[0]
	if len(call) < 3 || call[len(call)-2:] == "rr" {
		log.SetOutput(io.Discard)
		zerolog.TimeFieldFormat = time.RFC3339
		errLog = zerolog.New(os.Stderr).With().Timestamp().Logger()
	} else if call[len(call)-3:] == "rrv" {
		verbose = true
		log.SetOutput(new(logWriter))
	} else if call[len(call)-3:] == "rrd" {
		dump = true
		log.SetOutput(io.Discard)
	} else {
		lib.Bug("Unsupported executable name.")
	}
	if !dump {
		if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
			verbose = true
			log.SetOutput(new(logWriter))
		}
		log.Printf("rr %s %s", versionNumber, codeName)
	}
	isDir := lib.StatPath("directory")
	isFile := lib.StatPath("file")
	var sh strings.Builder

	var offset int
	var hostname string
	if len(os.Args) < 2 {
		if verbose {
			lib.Panic("Missing arguments.")
		} else {
			errLog.Error().Msg("Missing arguments")
			os.Exit(1)
		}
	}
	if strings.Contains(os.Args[1], "/") || strings.Contains(os.Args[1], ":") {
		offset = 1
		hostname = "local"
	} else {
		offset = 2
		hostname = os.Args[1]
	}
	if len(os.Args) < offset+1 {
		if verbose {
			lib.Panic("`namespace:script` not specified.")
		} else {
			errLog.Error().Msg("namespace:script not specified")
			os.Exit(1)
		}
	}
	s := strings.Split(os.Args[offset], "/")
	if len(s) < 2 {
		s = strings.Split(os.Args[offset], ":")
	}
	if len(s) < 2 {
		if verbose {
			lib.Panic("`namespace:script` not specified.")
		} else {
			errLog.Error().Msg("namespace:script not specified")
			os.Exit(1)
		}
	}
	namespace, script := s[0], s[1]
	if !isDir(namespace) {
		if verbose {
			lib.Panicf("`%s`(namespace) is not a directory.", namespace)
		} else {
			errLog.Error().Str("namespace", fmt.Sprintf("%s", namespace)).Msg("Namespace is not a directory")
			os.Exit(1)
		}
	}
	if !isDir(fmt.Sprintf("%s/%s", namespace, script)) {
		if verbose {
			lib.Panicf("`%s/%s` is not a diretory.", namespace, script)
		} else {
			errLog.Error().Str("namespace", fmt.Sprintf("%s", namespace)).Str("script", fmt.Sprintf("%s", script)).Msg("namespace/script is not a directory")
			os.Exit(1)
		}
	}
	if !isFile(fmt.Sprintf("%s/%s/%s", namespace, script, run)) {
		if verbose {
			lib.Panicf("`%s/%s/%s` actual script not found.", namespace, script, run)
		} else {
			errLog.Error().Str("namespace", fmt.Sprintf("%s", namespace)).Str("script", fmt.Sprintf("%s", script)).Msg("Actual script is missing")
			os.Exit(1)
		}
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
	if dump == true {
		fmt.Print(modscript)
		os.Exit(0)
	}
	const STDOUT = " ┌── stdout"
	const STDERR = " ┌── stderr"
	const STDDBG = " ┌── debug"
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
				var done func()
				if verbose {
					done = showSpinnerWhile(0)
				}
				ret, stdout, stderr, goerr := rargs.Run()
				if verbose {
					done()
				}
				if !ret {
					if !verbose {
						errLog.Error().Str("stdout", fmt.Sprintf("%s", stdout)).Str("stderr", fmt.Sprintf("%s", stderr)).Msg("Error copying files")
					} else {
						ho, bo, fo := output(stdout, hostname, STDOUT)
						he, be, fe := output(stderr, hostname, STDERR)
						hd, bd, fd := output(goerr, hostname, STDDBG)
						log.Printf("Failure copying files!\n%%s%ss%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
					}
					os.Exit(1)
				}
			}
		}
		rargs := lib.RunArgs{Exe: "sh", Args: []string{"-c", modscript}}
		var done func()
		if verbose {
			done = showSpinnerWhile(1)
		}
		ret, stdout, stderr, goerr := rargs.Run()
		if verbose {
			done()
		}
		ho, bo, fo := output(stdout, hostname, STDOUT)
		he, be, fe := output(stderr, hostname, STDERR)
		hd, bd, fd := output(goerr, hostname, STDDBG)
		if !ret {
			failed = true
			if !verbose {
				errLog.Error().Str("stdout", fmt.Sprintf("%s", stdout)).Str("stderr", fmt.Sprintf("%s", stderr)).Msg("Output")
			} else {
				log.Printf("Failure running script!\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
			}
		} else {
			if stdout != "" || stderr != "" || goerr != "" {
				log.Printf("Done. Output:\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
			}
		}
	} else if _, err := strconv.ParseInt(hostname, 10, 64); err == nil {
		destination := fmt.Sprintf("/proc/%s/root", hostname)
		for _, d := range []string{
			".files/",
			namespace + "/.files/",
			namespace + "/" + script + "/.files/",
		} {
			if isDir(d) {
				rsargs := lib.RunArgs{Exe: "rsync", Args: []string{"-q", "-a", d, destination}}
				var done func()
				if verbose {
					done = showSpinnerWhile(0)
				}
				ret, stdout, stderr, goerr := rsargs.Run()
				if verbose {
					done()
				}
				if !ret {
					if !verbose {
						errLog.Error().Str("stdout", fmt.Sprintf("%s", stdout)).Str("stderr", fmt.Sprintf("%s", stderr)).Msg("Error copying files")
					} else {
						ho, bo, fo := output(stdout, hostname, STDOUT)
						he, be, fe := output(stderr, hostname, STDERR)
						hd, bd, fd := output(goerr, hostname, STDDBG)
						log.Printf("Failure copying files!\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
					}
					os.Exit(1)
				}
			}
		}
		nsargs := lib.RunArgs{Exe: "nsenter", Args: []string{"-a", "-r", "-t", hostname, "sh", "-c", modscript}}
		var done func()
		if verbose {
			done = showSpinnerWhile(1)
		}
		ret, stdout, stderr, goerr := nsargs.Run()
		if verbose {
			done()
		}
		ho, bo, fo := output(stdout, hostname, STDOUT)
		he, be, fe := output(stderr, hostname, STDERR)
		hd, bd, fd := output(goerr, hostname, STDDBG)
		if !ret {
			failed = true
			if !verbose {
				errLog.Error().Str("stdout", fmt.Sprintf("%s", stdout)).Str("stderr", fmt.Sprintf("%s", stderr)).Msg("Output")
			} else {
				log.Printf("Failure running script!\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
			}
		} else {
			if stdout != "" || stderr != "" || goerr != "" {
				log.Printf("Done. Output:\n%s%s%s%s%s%s", ho, bo, fo, he, be, fe)
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
				if verbose {
					log.Printf("Hostname %s does not match remote host.", realhost)
				} else {
					errLog.Error().Str("hostname", fmt.Sprintf("%s", realhost)).Msg("Hostname does not match remote host")
				}
				os.Exit(1)
			} else {
				log.Printf("Remote host is %s\n", sshhost[0])
			}
		} else {
			if !verbose {
				errLog.Error().Str("host", realhost).Msg("Host does not exist or unreachable")
			} else {
				log.Printf("%s does not exist or unreachable.", realhost)
			}
			os.Exit(1)
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
					if verbose {
						log.Print("Cannot create temporary file.")
					} else {
						errLog.Error().Msg("Cannot create temporary file")
					}
					os.Exit(1)
				}
				defer os.Remove(tmpfile.Name())
				sftpc := []byte(fmt.Sprintf("lcd %s\ncd /\nput -fRp .\n bye\n", d))
				if _, err = tmpfile.Write(sftpc); err != nil {
					if verbose {
						log.Print("Failed to write to temporary file.")
					} else {
						errLog.Error().Msg("Failed to write to temporary file")
					}
					os.Exit(1)
				}
				tmpfile.Close()
				sftpa := lib.RunArgs{Exe: "sftp", Args: []string{"-C", "-b", tmpfile.Name(), hostname}, Env: sshenv}
				var done func()
				if verbose {
					done = showSpinnerWhile(0)
				}
				ret, _, _, _ := sftpa.Run()
				if verbose {
					done()
				}
				if !ret {
					if verbose {
						log.Print("Running sftp failed.")
					} else {
						errLog.Error().Msg("Running sftp failed.")
					}
					os.Exit(1)
				}
				os.Remove(tmpfile.Name())
			}
		}
		log.Println("Running script...")
		sshb := lib.RunArgs{Exe: "ssh", Args: []string{"-T", "-a", "-x", "-C", hostname}, Env: sshenv,
			Stdin: []byte(modscript)}
		var done func()
		if verbose {
			done = showSpinnerWhile(1)
		}
		ret, stdout, stderr, goerr := sshb.Run()
		if verbose {
			done()
		}
		ho, bo, fo := output(stdout, hostname, STDOUT)
		he, be, fe := output(stderr, hostname, STDERR)
		hd, bd, fd := output(goerr, hostname, STDDBG)
		if !ret {
			failed = true
			if !verbose {
				errLog.Error().Str("stdout", fmt.Sprintf("%s", stdout)).Str("stderr", fmt.Sprintf("%s", stderr)).Msg("Error copying files")
			} else {
				log.Printf("Failure running script!\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
			}
		} else {
			if stdout != "" || stderr != "" || goerr != "" {
				log.Printf("Done. Output:\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
			}
		}
	}
	if !failed {
		log.Printf("Total run time: %s. All OK.", time.Since(start))
		os.Exit(0)
	} else {
		if verbose {
			log.Printf("Total run time: %s. Something went wrong.", time.Since(start))
		} else {
			errLog.Error().Str("elapsed", fmt.Sprintf("%s", time.Since(start))).Msg("Something went wrong.")
		}
		os.Exit(1)
	}
}
