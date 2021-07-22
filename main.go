package main

import (
	"bytes"
	"fmt"
	"hash/maphash"
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
	ulid "github.com/tongson/rr/external/ulid"
)

var start = time.Now()

const versionNumber = "0.10.0"
const codeName = "\"Kilowatt Triceps\""
const run = "script"
const STDOUT = " ┌─ stdout"
const STDERR = " ┌─ stderr"
const STDDBG = " ┌─ debug"
const FOOTER = " └─"
const PIPEST = "│"

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
	fmt.Printf("\033[38;2;85;85;85m%s\033[0m", time.Now().Format(time.RFC1123Z))
	return fmt.Print(" " + string(bytes))
}

func output(o string, h string, c string) (string, string, string) {
	rh := ""
	rb := ""
	rf := ""
	if o != "" {
		rh = fmt.Sprintf(" %s%s\n", h, c)
		rb = fmt.Sprintf("%s\n", lib.PipeStr(h, PIPEST, o))
		rf = fmt.Sprintf(" %s%s\n", h, FOOTER)
	}
	return rh, rb, rf
}

func main() {
	var serrLog zerolog.Logger
	var jsonLog zerolog.Logger
	var console bool = false
	var failed bool = false
	var dump bool = false
	runtime.MemProfileRate = 0
	defer lib.RecoverPanic()
	log.SetFlags(0)
	if call := os.Args[0]; len(call) < 3 || call[len(call)-2:] == "rr" {
		log.SetOutput(io.Discard)
		zerolog.TimeFieldFormat = time.RFC3339
		jsonFile, _ := os.OpenFile("rr.json", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
		jsonLog = zerolog.New(jsonFile).With().Timestamp().Logger()
		serrLog = zerolog.New(os.Stderr).With().Timestamp().Logger()
	} else if call[len(call)-3:] == "rrv" {
		console = true
		log.SetOutput(new(logWriter))
	} else if call[len(call)-3:] == "rrd" {
		dump = true
		log.SetOutput(io.Discard)
	} else {
		lib.Bug("Unsupported executable name.")
	}
	if !dump {
		if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
			console = true
			log.SetOutput(new(logWriter))
		}
		log.Printf("rr %s %s", versionNumber, codeName)
	}
	isDir := lib.StatPath("directory")
	isFile := lib.StatPath("file")
	var id string
	{
		b := []byte(strconv.FormatUint(new(maphash.Hash).Sum64(), 10))
		e := bytes.NewReader(b)
		var uid ulid.ULID
		uid.SetTime(ulid.Timestamp(time.Now()))
		io.ReadFull(e, uid[6:])
		id = uid.String()
	}
	var offset int
	var hostname string
	if len(os.Args) < 2 {
		if console {
			lib.Panic("Missing arguments.")
		} else {
			serrLog.Error().Msg("Missing arguments")
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
	
	// Handle readmes
	{
		isReadme := func(s string) (bool, string) {
			s = strings.TrimSuffix(s, "/")
			s = strings.TrimSuffix(s, ":")
			match, _ := lib.FileGlob(fmt.Sprintf("%s/README*", s))
			for _, m := range match {
				if isFile(m) {
					return true, m
				}
			}
			return false, ""
		}
		printReadme := func(s string) {
			sz := len(s)
			line := strings.Repeat("─", sz+2)
			fmt.Print(fmt.Sprintf("%s┐\n", line))
			if console {
				fmt.Printf(" \033[37;1m%s\033[0m │\n", s)
			} else {
				fmt.Printf(" %s │\n", s)
			}
			fmt.Print(fmt.Sprintf("%s┘\n", line))
			if console {
				for _, each := range lib.FileLines(s) {
					fmt.Printf(" \033[38;2;85;85;85m⋮\033[0m %s\n", each)
				}
				fmt.Printf("\n")
			} else {
				fmt.Print(lib.FileRead(s))
			}
		}
		if found1, readme1 := isReadme(os.Args[1]); found1 == true && readme1 != "" {
			log.Print("Showing the readme...")
			printReadme(readme1)
			os.Exit(0)
		} else if len(os.Args) > 2 {
			if found2, readme2 := isReadme(os.Args[2]); found2 == true && readme2 != "" {
				log.Print("Showing the readme...")
				printReadme(readme2)
				os.Exit(0)
			}
		}
	}
	if len(os.Args) < offset+1 {
		if console {
			lib.Panic("`namespace:script` not specified.")
		} else {
			serrLog.Error().Msg("namespace:script not specified")
			os.Exit(1)
		}
	}
	var sh strings.Builder
	var namespace string
	var script string
	{
		var s []string
		// Old behavior. Allowed hacky tab completion by replacing the '/' with ':'.
		// Ditched because of the README feature.
		// s := strings.Split(os.Args[offset], "/")
		if len(s) < 2 {
			s = strings.Split(os.Args[offset], ":")
		}
		if len(s) < 2 {
			if console {
				lib.Panic("`namespace:script` not specified.")
			} else {
				serrLog.Error().Msg("namespace:script not specified")
				os.Exit(1)
			}
		}
		namespace, script = s[0], s[1]
		if !isDir(namespace) {
			if console {
				lib.Panicf("`%s`(namespace) is not a directory.", namespace)
			} else {
				serrLog.Error().Str("namespace", fmt.Sprintf("%s", namespace)).Msg("Namespace is not a directory")
				os.Exit(1)
			}
		}
		if !isDir(fmt.Sprintf("%s/%s", namespace, script)) {
			if console {
				lib.Panicf("`%s/%s` is not a diretory.", namespace, script)
			} else {
				serrLog.Error().Str("namespace", fmt.Sprintf("%s", namespace)).Str("script", fmt.Sprintf("%s", script)).Msg("namespace/script is not a directory")
				os.Exit(1)
			}
		}
		if !isFile(fmt.Sprintf("%s/%s/%s", namespace, script, run)) {
			if console {
				lib.Panicf("`%s/%s/%s` actual script not found.", namespace, script, run)
			} else {
				serrLog.Error().Str("namespace", fmt.Sprintf("%s", namespace)).Str("script", fmt.Sprintf("%s", script)).Msg("Actual script is missing")
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
	}
	modscript := sh.String()
	if dump == true {
		fmt.Print(modscript)
		os.Exit(0)
	}
	log.Printf("Running %s:%s via %s...", namespace, script, hostname)
	if console {
		jsonLog.Debug().Str("id", id).Str("namespace", namespace).Str("script", script).Str("target", hostname).Msg("running")
	}
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
				log.Printf("Copying %s...", d)
				if console {
					jsonLog.Debug().Str("id", id).Str("directory", d).Msg("copying")
				}
				rargs := lib.RunArgs{Exe: "sh", Args: []string{"-c", fmt.Sprintf(untar, d)}}
				var done func()
				if console {
					done = showSpinnerWhile(0)
				}
				ret, stdout, stderr, goerr := rargs.Run()
				if console {
					done()
				}
				if op := "copy"; !ret {
					if !console {
						serrLog.Error().Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(op)
					} else {
						jsonLog.Error().Str("id", id).Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(op)
						ho, bo, fo := output(stdout, hostname, STDOUT)
						he, be, fe := output(stderr, hostname, STDERR)
						hd, bd, fd := output(goerr, hostname, STDDBG)
						log.Printf("Error encountered.\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
						log.Printf("Failure copying files!")
					}
					os.Exit(1)
				} else {
					if console {
						jsonLog.Info().Str("id", id).Str("result", "success").Msg(op)
					}
					log.Printf("Successfully copied files")
				}
			}
		}
		log.Printf("Running %s...", script)
		if console {
			jsonLog.Debug().Str("id", id).Str("script", script).Msg("running")
		}
		rargs := lib.RunArgs{Exe: "sh", Args: []string{"-c", modscript}}
		var done func()
		if console {
			done = showSpinnerWhile(1)
		}
		ret, stdout, stderr, goerr := rargs.Run()
		if console {
			done()
		}
		ho, bo, fo := output(stdout, hostname, STDOUT)
		he, be, fe := output(stderr, hostname, STDERR)
		hd, bd, fd := output(goerr, hostname, STDDBG)
		if op := "run"; !ret {
			failed = true
			if !console {
				serrLog.Error().Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(op)
			} else {
				jsonLog.Error().Str("id", id).Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(op)
				log.Printf("Failure running script!\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
			}
		} else {
			if console {
				jsonLog.Info().Str("id", id).Str("result", "success").Msg(op)
			}
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
				log.Printf("Copying %s...", d)
				if console {
					jsonLog.Debug().Str("id", id).Str("directory", d).Msg("copying")
				}
				rsargs := lib.RunArgs{Exe: "rsync", Args: []string{"-q", "-a", d, destination}}
				var done func()
				if console {
					done = showSpinnerWhile(0)
				}
				ret, stdout, stderr, goerr := rsargs.Run()
				if console {
					done()
				}
				if op := "copy"; !ret {
					if !console {
						serrLog.Error().Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(op)
					} else {
						jsonLog.Error().Str("id", id).Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(op)
						ho, bo, fo := output(stdout, hostname, STDOUT)
						he, be, fe := output(stderr, hostname, STDERR)
						hd, bd, fd := output(goerr, hostname, STDDBG)
						log.Printf("Error encountered.\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
						log.Printf("Failure copying files!")
					}
					os.Exit(1)
				} else {
					if console {
						jsonLog.Info().Str("id", id).Str("result", "success").Msg(op)
					}
					log.Printf("Successfully copied files")
				}
			}
		}
		log.Printf("Running %s...", script)
		if console {
			jsonLog.Debug().Str("id", id).Str("script", script).Msg("running")
		}
		nsargs := lib.RunArgs{Exe: "nsenter", Args: []string{"-a", "-r", "-t", hostname, "sh", "-c", modscript}}
		var done func()
		if console {
			done = showSpinnerWhile(1)
		}
		ret, stdout, stderr, goerr := nsargs.Run()
		if console {
			done()
		}
		ho, bo, fo := output(stdout, hostname, STDOUT)
		he, be, fe := output(stderr, hostname, STDERR)
		hd, bd, fd := output(goerr, hostname, STDDBG)
		if op := "run"; !ret {
			failed = true
			if !console {
				serrLog.Error().Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(op)
			} else {
				jsonLog.Error().Str("id", id).Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(op)
				log.Printf("Failure running script!\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
			}
		} else {
			if console {
				jsonLog.Info().Str("id", id).Str("result", "success").Msg(op)
			}
			if stdout != "" || stderr != "" || goerr != "" {
				log.Printf("Done. Output:\n%s%s%s%s%s%s", ho, bo, fo, he, be, fe)
			}
		}
	} else {
		var realhost string
		if rh := strings.Split(hostname, "@"); len(rh) == 1 {
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
				if console {
					jsonLog.Error().Str("id", id).Str("hostname", realhost).Msg("Hostname does not match remote host")
					log.Printf("Hostname %s does not match remote host.", realhost)
				} else {
					serrLog.Error().Str("hostname", realhost).Msg("Hostname does not match remote host")
				}
				os.Exit(1)
			} else {
				log.Printf("Remote host is %s\n", sshhost[0])
			}
		} else {
			if !console {
				serrLog.Error().Str("host", realhost).Msg("Host does not exist or unreachable")
			} else {
				jsonLog.Error().Str("id", id).Str("host", realhost).Msg("Host does not exist or unreachable")
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
				if console {
					jsonLog.Debug().Str("id", id).Str("directory", d).Msg("copying")
				}
				log.Printf("Copying %s to %s...", d, realhost)
				tmpfile, err := os.CreateTemp(os.TempDir(), "_rr")
				if err != nil {
					if console {
						jsonLog.Error().Str("id", id).Msg("Cannot create temporary file.")
						log.Print("Cannot create temporary file.")
					} else {
						serrLog.Error().Msg("Cannot create temporary file")
					}
					os.Exit(1)
				}
				defer os.Remove(tmpfile.Name())
				sftpc := []byte(fmt.Sprintf("lcd %s\ncd /\nput -fRp .\n bye\n", d))
				if _, err = tmpfile.Write(sftpc); err != nil {
					if console {
						jsonLog.Error().Str("id", id).Msg("Failed to write to temporary file.")
						log.Print("Failed to write to temporary file.")
					} else {
						serrLog.Error().Msg("Failed to write to temporary file")
					}
					os.Exit(1)
				}
				tmpfile.Close()
				sftpa := lib.RunArgs{Exe: "sftp", Args: []string{"-C", "-b", tmpfile.Name(), hostname}, Env: sshenv}
				var done func()
				if console {
					done = showSpinnerWhile(0)
				}
				ret, stdout, stderr, goerr := sftpa.Run()
				os.Remove(tmpfile.Name())
				if console {
					done()
				}
				if op := "copy"; !ret {
					if !console {
						serrLog.Error().Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(op)
					} else {
						jsonLog.Error().Str("id", id).Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(op)
						ho, bo, fo := output(stdout, hostname, STDOUT)
						he, be, fe := output(stderr, hostname, STDERR)
						hd, bd, fd := output(goerr, hostname, STDDBG)
						log.Printf("Error encountered.\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
						log.Printf("Failure copying files!")
					}
					os.Exit(1)
				} else {
					if console {
						jsonLog.Info().Str("id", id).Str("result", "success").Msg(op)
					}
					log.Printf("Successfully copied files")
				}
			}
		}
		log.Printf("Running %s...", script)
		if console {
			jsonLog.Debug().Str("id", id).Str("script", script).Msg("running")
		}
		sshb := lib.RunArgs{Exe: "ssh", Args: []string{"-T", "-a", "-x", "-C", hostname}, Env: sshenv,
			Stdin: []byte(modscript)}
		var done func()
		if console {
			done = showSpinnerWhile(1)
		}
		ret, stdout, stderr, goerr := sshb.Run()
		if console {
			done()
		}
		ho, bo, fo := output(stdout, hostname, STDOUT)
		he, be, fe := output(stderr, hostname, STDERR)
		hd, bd, fd := output(goerr, hostname, STDDBG)
		if op := "run"; !ret {
			failed = true
			if !console {
				serrLog.Error().Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(op)
			} else {
				jsonLog.Error().Str("id", id).Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(op)
				log.Printf("Failure running script!\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
			}
		} else {
			if console {
				jsonLog.Info().Str("id", id).Str("result", "success").Msg(op)
			}
			if stdout != "" || stderr != "" || goerr != "" {
				log.Printf("Done. Output:\n%s%s%s%s%s%s", ho, bo, fo, he, be, fe)
			}
		}
	}
	if tm := fmt.Sprintf("%s", time.Since(start)); !failed {
		if console {
			jsonLog.Debug().Str("id", id).Str("elapsed", tm).Msg("success")
		}
		log.Printf("Total run time: %s. All OK.", time.Since(start))
		os.Exit(0)
	} else {
		if console {
			jsonLog.Debug().Str("id", id).Str("elapsed", tm).Msg("failed")
			log.Printf("Total run time: %s. Something went wrong.", time.Since(start))
		} else {
			serrLog.Debug().Str("elapsed", tm).Msg("failed")
		}
		os.Exit(1)
	}
}
