package main

import (
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"hash/maphash"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	zerolog "github.com/rs/zerolog"
	lib "github.com/tongson/gl"
	spin "github.com/tongson/rr/external/go-spin"
)

var start = time.Now()

const VersionNumber = "0.16.0"
const CodeName = "\"Tainted Battalion\""

const OP = "task"
const RUN = "script"
const LOG = "rr.json"
const DOC = "README"
const INTERP = "shell"
const HOSTS = "rr.hosts"

const STDOUT = " ┌─ stdout"
const STDERR = " ┌─ stderr"
const STDDBG = " ┌─ debug"
const FOOTER = " └─"
const PIPEST = "│"

type logWriter struct {
}

// https://gist.github.com/jlinoff/e8e26b4ffa38d379c7f1891fd174a6d0
func getPassword(prompt string) string {
	// Get the initial state of the terminal.
	initialTermState, e1 := terminal.GetState(syscall.Stdin)
	if e1 != nil {
		panic(e1)
	}

	// Restore it in the event of an interrupt.
	// CITATION: Konstantin Shaposhnikov - https://groups.google.com/forum/#!topic/golang-nuts/kTVAbtee9UA
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		_ = terminal.Restore(syscall.Stdin, initialTermState)
		os.Exit(1)
	}()

	// Now get the password.
	fmt.Print(prompt)
	p, err := terminal.ReadPassword(syscall.Stdin)
	fmt.Println("")
	if err != nil {
		panic(err)
	}

	// Stop looking for ^C on the channel.
	signal.Stop(c)

	// Return the password as a string.
	return string(p)
}

func showSpinnerWhile(s int) func() {
	spinner := spin.New()
	switch s {
	case 0:
		spinner.Set(spin.Spin26)
	default:
		spinner.Set(spin.Box1)
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
		close(done)
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

func sshexec(sudo bool, script string, hostname string, id string, interp string, sshconfig string, password string) (bool, string, string, string) {
	tmps := fmt.Sprintf("./.__rr.scr.%s", id)
	sshenv := []string{"LC_ALL=C"}
	var ssha lib.RunArgs
	var sshb lib.RunArgs
	var sshc lib.RunArgs
	if sshconfig == "" {
		args := []string{"-a", "-T", "-x", hostname, fmt.Sprintf("cat - > %s", tmps)}
		ssha = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte(script)}
	} else {
		args := []string{"-F", sshconfig, "-a", "-T", "-x", hostname, fmt.Sprintf("cat - > %s", tmps)}
		ssha = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte(script)}
	}
	if ret, stdout, stderr, goerr := ssha.Run(); !ret {
		return ret, stdout, stderr, goerr
	}
	var ret bool
	var stdout string
	var stderr string
	var goerr string
	if sshconfig == "" {
		if !sudo {
			args := []string{"-a", "-T", "-x", hostname, interp, tmps}
			sshb = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte(password)}
		} else {
			args := []string{"-a", "-T", "-x", hostname, "sudo", "-k", "--prompt=\"\"", "-S", "-s", "--", interp, tmps}
			sshb = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte(password)}
		}
	} else {
		if !sudo {
			args := []string{"-F", sshconfig, "-a", "-T", "-x", hostname, interp, tmps}
			sshb = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte(password)}
		} else {
			args := []string{"-F", sshconfig, "-a", "-T", "-x", hostname, "sudo", "-k", "--prompt=\"\"", "-S", "-s", "--", interp, tmps}
			sshb = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte(password)}
		}
	}
	ret, stdout, stderr, goerr = sshb.Run()
	if !ret {
		return ret, stdout, stderr, goerr
	}
	if sshconfig == "" {
		args := []string{"-a", "-T", "-x", hostname, fmt.Sprintf("rm -f %s", tmps)}
		sshc = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv}
	} else {
		args := []string{"-F", sshconfig, "-a", "-T", "-x", hostname, fmt.Sprintf("rm -f %s", tmps)}
		sshc = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv}
	}
	if xret, xstdout, xstderr, xgoerr := sshc.Run(); !xret {
		return xret, xstdout, xstderr, xgoerr
	} else {
		return ret, stdout, stderr, goerr
	}
}

func sudocopy(dir string, hostname string, id string, interp string, sshconfig string, password string) (bool, string, string, string) {
	tmpd := fmt.Sprintf(".__rr.dir.%s", id)
	tmpf := fmt.Sprintf("./.__rr.tar.%s", id)
	tarcmd := `
	set -efu
	LC_ALL=C
	unset IFS
	tar -C %s -cf - . | tar -C / --overwrite --no-same-owner -ompxf -
	rm -rf %s
	rm -f %s
	`
	tarexec := fmt.Sprintf(tarcmd, tmpd, tmpd, tmpf)
	sshenv := []string{"LC_ALL=C"}
	var untar1 lib.RunArgs
	if sshconfig == "" {
		untar1 = lib.RunArgs{Exe: "ssh", Args: []string{"-a", "-T", "-x", hostname, fmt.Sprintf("cat - > %s", tmpf)}, Env: sshenv, Stdin: []byte(tarexec)}
	} else {
		args := []string{"-F", sshconfig, "-a", "-T", "-x", hostname, fmt.Sprintf("cat - > %s", tmpf)}
		untar1 = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte(tarexec)}
	}
	if ret, stdout, stderr, goerr := untar1.Run(); !ret {
		return ret, stdout, stderr, goerr
	}
	untarDefault := `
	RRHOST="%s"
	RRSRC="%s"
	RRDEST="%s"
	RRSCRIPT="%s"
	ssh -T -x "$RRHOST" mkdir "$RRDEST"
	tar -C "$RRSRC" -czf - . | ssh -a -T -x "$RRHOST" tar -C "$RRDEST" --no-same-owner -omxpzf -
	`
	untarConfig := `
	RRHOST="%s"
	RRSRC="%s"
	RRDEST="%s"
	RRCONFIG="%s"
	RRSCRIPT="%s"
	ssh -F "$RRCONFIG" -T -x "$RRHOST" mkdir "$RRDEST"
	tar -C "$RRSRC" -czf - . | ssh -F "$RRCONFIG" -a -T -x "$RRHOST" tar -C "$RRDEST" --no-same-owner -omxpzf -
	`
	tarenv := []string{"LC_ALL=C", "PATH=/bin:/usr/bin"}
	var untar2 lib.RunArgs
	if sshconfig == "" {
		untar2 = lib.RunArgs{Exe: interp, Args: []string{"-c", fmt.Sprintf(untarDefault, hostname, dir, tmpd, tmpf)}, Env: tarenv}
	} else {
		untar2 = lib.RunArgs{Exe: interp, Args: []string{"-c", fmt.Sprintf(untarConfig, hostname, dir, tmpd, sshconfig, tmpf)}, Env: tarenv}
	}
	if ret, stdout, stderr, goerr := untar2.Run(); !ret {
		return ret, stdout, stderr, goerr
	}
	var untar3 lib.RunArgs
	if sshconfig == "" {
		args := []string{"-a", "-T", "-x", hostname, "sudo", "-k", "--prompt=\"\"", "-S", "-s", "--", interp, tmpf}
		untar3 = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte(password)}
	} else {
		args := []string{"-F", sshconfig, "-a", "-T", "-x", hostname, "sudo", "-k", "--prompt=\"\"", "-S", "-s", "--", interp, tmpf}
		untar3 = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte(password)}
	}
	return untar3.Run()
}

func quickcopy(dir string, hostname string, interp string, sshconfig string) (bool, string, string, string) {
	untarDefault := `
	set -o errexit -o nounset -o noglob
	tar -C %s -czf - . | ssh -a -T -x %s tar -C / --overwrite --no-same-owner -omxpzf -
	`
	untarConfig := `
	tar -C %s -czf - . | ssh -F %s -a -T -x %s tar -C / --overwrite --no-same-owner -omxpzf -
	`
	tarenv := []string{"LC_ALL=C", "PATH=/bin:/usr/bin"}
	var untar lib.RunArgs
	if sshconfig == "" {
		untar = lib.RunArgs{Exe: interp, Args: []string{"-c", fmt.Sprintf(untarDefault, dir, hostname)}, Env: tarenv}
	} else {
		untar = lib.RunArgs{Exe: interp, Args: []string{"-c", fmt.Sprintf(untarConfig, dir, sshconfig, hostname)}, Env: tarenv}
	}
	return untar.Run()
}

func main() {
	var serrLog zerolog.Logger
	var jsonLog zerolog.Logger
	var console bool = false
	var failed bool = false
	var dump bool = false
	var logger bool = false
	var sudo bool = false
	var sudoPassword string = ""
	var nopasswd bool = false
	var sshconfig string = ""
	runtime.MemProfileRate = 0
	defer lib.RecoverPanic()
	log.SetFlags(0)
	if call := os.Args[0]; len(call) < 3 || call[len(call)-2:] == "rr" {
		log.SetOutput(io.Discard)
		logger = true
	} else if call[len(call)-3:] == "rrv" {
		console = true
		log.SetOutput(new(logWriter))
	} else if call[len(call)-3:] == "rrd" {
		dump = true
		log.SetOutput(io.Discard)
	} else if call[len(call)-3:] == "rrs" {
		sudo = true
		logger = true
	} else if call[len(call)-3:] == "rru" {
		sudo = true
		nopasswd = true
		logger = true
	} else {
		lib.Bug("Unsupported executable name. Valid: `rr(local/ssh)`, `rrs(ssh+sudo)`, `rrd(dump)`, `rrv(force verbose)`")
	}
	if !dump {
		if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
			console = true
			log.SetOutput(new(logWriter))
		}
		log.Printf("rr %s %s", VersionNumber, CodeName)
	}
	if logger {
		zerolog.TimeFieldFormat = time.RFC3339
		jsonFile, _ := os.OpenFile(LOG, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
		jsonLog = zerolog.New(jsonFile).With().Timestamp().Logger()
		serrLog = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}
	isDir := lib.StatPath("directory")
	isFile := lib.StatPath("file")
	var id string
	{
		h := new(maphash.Hash)
		id = fmt.Sprintf("%016X", h.Sum64())
	}
	var offset int
	var hostname string
	if len(os.Args) < 2 {
		if console {
			lib.Panic("Missing arguments.")
		} else {
			serrLog.Fatal().Msg("Missing arguments")
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
			match, _ := lib.FileGlob(fmt.Sprintf("%s/%s*", s, DOC))
			for _, m := range match {
				if isFile(m) {
					return true, m
				}
			}
			return false, ""
		}
		printReadme := func(s string) {
			ps := strings.Split(s, "/")
			s1 := ps[0]
			var s2 string
			var s3 string
			if len(ps) == 2 {
				s2 = "*"
				s3 = ps[1]
			} else {
				s2 = ps[1]
				s3 = ps[2]
			}
			pps := fmt.Sprintf("rr %s:%s (%s)", s1, s2, s3)
			sz := len(pps)
			line := strings.Repeat("─", sz+2)
			fmt.Print(fmt.Sprintf("%s┐\n", line))
			if console {
				fmt.Printf(" \033[37;1m%s\033[0m │\n", pps)
			} else {
				fmt.Printf(" %s │\n", pps)
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
			log.Print("Showing README…")
			printReadme(readme1)
			os.Exit(0)
		} else if len(os.Args) > 2 {
			if found2, readme2 := isReadme(os.Args[2]); found2 == true && readme2 != "" {
				log.Print("Showing README…")
				printReadme(readme2)
				os.Exit(0)
			}
		}
	}
	if len(os.Args) < offset+1 {
		if console {
			lib.Panic("`namespace:script` not specified.")
		} else {
			serrLog.Fatal().Msg("namespace:script not specified")
			os.Exit(1)
		}
	}
	var sh strings.Builder
	var namespace string
	var script string
	var code string
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
				serrLog.Fatal().Msg("namespace:script not specified")
				os.Exit(1)
			}
		}
		namespace, script = s[0], s[1]
		if !isDir(namespace) {
			if console {
				lib.Panicf("`%s`(namespace) is not a directory.", namespace)
			} else {
				serrLog.Fatal().Str("namespace", fmt.Sprintf("%s", namespace)).Msg("Namespace is not a directory")
				os.Exit(1)
			}
		}
		if !isDir(fmt.Sprintf("%s/%s", namespace, script)) {
			if console {
				lib.Panicf("`%s/%s` is not a diretory.", namespace, script)
			} else {
				serrLog.Fatal().Str("namespace", fmt.Sprintf("%s", namespace)).Str("script", fmt.Sprintf("%s", script)).Msg("namespace/script is not a directory")
				os.Exit(1)
			}
		}
		if !isFile(fmt.Sprintf("%s/%s/%s", namespace, script, RUN)) {
			if console {
				lib.Panicf("`%s/%s/%s` actual script not found.", namespace, script, RUN)
			} else {
				serrLog.Fatal().Str("namespace", fmt.Sprintf("%s", namespace)).Str("script", fmt.Sprintf("%s", script)).Msg("Actual script is missing")
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
		if sudo {
			if !nopasswd {
				sudoPassword = fmt.Sprintf("%s\n", getPassword("sudo password: "))
			}
		}
		//Pass environment variables with `rr` prefix
		for _, e := range os.Environ() {
			if strings.HasPrefix(e, "rr__") {
				sh.WriteString("export " + strings.TrimPrefix(e, "rr__") + "\n")
			}
		}
		arguments = lib.InsertStr(arguments, "set --", 0)
		sh.WriteString(strings.Join(arguments, " "))
		code = lib.FileRead(namespace + "/" + script + "/" + RUN)
		sh.WriteString("\n" + code)
	}
	modscript := sh.String()
	if dump == true {
		fmt.Print(modscript)
		os.Exit(0)
	}
	interp := lib.FileRead(fmt.Sprintf("%s/%s/%s", namespace, script, INTERP))
	interp = strings.TrimSuffix(interp, "\n")
	if interp == "" {
		interp = "sh"
	}
	op := lib.FileRead(fmt.Sprintf("%s/%s/%s", namespace, script, OP))
	op = strings.Split(op, "\n")[0]
	if op == "" {
		op = "UNDEFINED"
	}
	log.Printf("Running %s:%s via %s…", namespace, script, hostname)
	if console {
		jsonLog.Info().Str("app", "rr").Str("id", id).Str("namespace", namespace).Str("script", script).Str("target", hostname).Msg(op)
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
				log.Printf("Copying %s…", d)
				if console {
					jsonLog.Debug().Str("app", "rr").Str("id", id).Str("directory", d).Msg("copying")
				}
				rargs := lib.RunArgs{Exe: interp, Args: []string{"-c", fmt.Sprintf(untar, d)}}
				var done func()
				if console {
					done = showSpinnerWhile(0)
				}
				ret, stdout, stderr, goerr := rargs.Run()
				b64so := base64.StdEncoding.EncodeToString([]byte(stdout))
				b64se := base64.StdEncoding.EncodeToString([]byte(stderr))
				if console {
					done()
				}
				if step := "copy"; !ret {
					if !console {
						serrLog.Error().Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(step)
					} else {
						jsonLog.Error().Str("app", "rr").Str("id", id).Str("stdout", b64so).Str("stderr", b64se).Str("error", goerr).Msg(step)
						ho, bo, fo := output(stdout, hostname, STDOUT)
						he, be, fe := output(stderr, hostname, STDERR)
						hd, bd, fd := output(goerr, hostname, STDDBG)
						log.Printf("Error encountered.\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
						log.Printf("Failure copying files!")
					}
					os.Exit(1)
				} else {
					if console {
						jsonLog.Debug().Str("app", "rr").Str("id", id).Str("stdout", b64so).Str("stderr", b64se).Str("error", goerr).Msg(step)
						jsonLog.Info().Str("app", "rr").Str("id", id).Str("result", "success").Msg(step)
					}
					log.Printf("Successfully copied files")
				}
			}
		}
		if op == "UNDEFINED" {
			log.Printf("Running %s…", script)
			if console {
				jsonLog.Debug().Str("app", "rr").Str("id", id).Str("script", script).Msg("running")
			}
		} else {
			msgop := strings.TrimSuffix(op, "\n")
			log.Printf("%s…", msgop)
			if console {
				jsonLog.Debug().Str("app", "rr").Str("id", id).Str("script", script).Msg(msgop)
			}
		}
		rargs := lib.RunArgs{Exe: interp, Args: []string{"-c", modscript}}
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
		b64so := base64.StdEncoding.EncodeToString([]byte(stdout))
		b64se := base64.StdEncoding.EncodeToString([]byte(stderr))
		b64sc := base64.StdEncoding.EncodeToString([]byte(code))
		if !ret {
			failed = true
			if !console {
				serrLog.Error().Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(op)
			} else {
				jsonLog.Error().Str("app", "rr").Str("id", id).Str("code", b64sc).Str("stdout", b64so).Str("stderr", b64se).Str("error", goerr).Msg(op)
				log.Printf("Failure running script!\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
			}
		} else {
			if console {
				jsonLog.Debug().Str("app", "rr").Str("id", id).Str("code", b64sc).Str("stdout", b64so).Str("stderr", b64se).Str("error", goerr).Msg(op)
				jsonLog.Info().Str("app", "rr").Str("id", id).Str("result", "success").Msg(op)
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
				log.Printf("Copying %s…", d)
				if console {
					jsonLog.Debug().Str("app", "rr").Str("id", id).Str("directory", d).Msg("copying")
				}
				tarenv := []string{"LC_ALL=C", "PATH=/bin:/usr/bin"}
				untar := `
				tar -C %s -cf - . | tar -C %s --no-same-owner --overwrite -omxpf -
				`
				rsargs := lib.RunArgs{Exe: interp, Args: []string{"-c", fmt.Sprintf(untar, d, destination)}, Env: tarenv}
				var done func()
				if console {
					done = showSpinnerWhile(0)
				}
				ret, stdout, stderr, goerr := rsargs.Run()
				b64so := base64.StdEncoding.EncodeToString([]byte(stdout))
				b64se := base64.StdEncoding.EncodeToString([]byte(stderr))
				if console {
					done()
				}
				if step := "copy"; !ret {
					if !console {
						serrLog.Error().Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(step)
					} else {
						jsonLog.Error().Str("app", "rr").Str("id", id).Str("stdout", b64so).Str("stderr", b64se).Str("error", goerr).Msg(step)
						ho, bo, fo := output(stdout, hostname, STDOUT)
						he, be, fe := output(stderr, hostname, STDERR)
						hd, bd, fd := output(goerr, hostname, STDDBG)
						log.Printf("Error encountered.\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
						log.Printf("Failure copying files!")
					}
					os.Exit(1)
				} else {
					if console {
						jsonLog.Debug().Str("app", "rr").Str("id", id).Str("stdout", b64so).Str("stderr", b64se).Str("error", goerr).Msg(step)
						jsonLog.Info().Str("app", "rr").Str("id", id).Str("result", "success").Msg(step)
					}
					log.Printf("Successfully copied files")
				}
			}
		}
		log.Printf("Running %s…", script)
		if console {
			jsonLog.Debug().Str("app", "rr").Str("id", id).Str("script", script).Msg("running")
		}
		nsargs := lib.RunArgs{Exe: "nsenter", Args: []string{"-a", "-r", "-t", hostname, interp, "-c", modscript}}
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
		b64so := base64.StdEncoding.EncodeToString([]byte(stdout))
		b64se := base64.StdEncoding.EncodeToString([]byte(stderr))
		b64sc := base64.StdEncoding.EncodeToString([]byte(code))
		if !ret {
			failed = true
			if !console {
				serrLog.Error().Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(op)
			} else {
				jsonLog.Error().Str("app", "rr").Str("id", id).Str("code", b64sc).Str("stdout", b64so).Str("stderr", b64se).Str("error", goerr).Msg(op)
				log.Printf("Failure running script!\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
			}
		} else {
			if console {
				jsonLog.Debug().Str("app", "rr").Str("id", id).Str("code", b64sc).Str("stdout", b64so).Str("stderr", b64se).Str("error", goerr).Msg(op)
				jsonLog.Info().Str("app", "rr").Str("id", id).Str("result", "success").Msg(op)
			}
			if stdout != "" || stderr != "" || goerr != "" {
				log.Printf("Done. Output:\n%s%s%s%s%s%s", ho, bo, fo, he, be, fe)
			}
		}
	} else {
		if lib.IsFile(HOSTS) {
			sshconfig = HOSTS
		}
		var realhost string
		if rh := strings.Split(hostname, "@"); len(rh) == 1 {
			realhost = hostname
		} else {
			realhost = rh[1]
		}
		{
			var done func()
			if console {
				done = showSpinnerWhile(0)
			}
			sshenv := []string{"LC_ALL=C"}
			var ssha lib.RunArgs
			if sshconfig == "" {
				ssha = lib.RunArgs{Exe: "ssh", Args: []string{"-T", "-x", hostname, "uname -n"}, Env: sshenv}
			} else {
				ssha = lib.RunArgs{Exe: "ssh", Args: []string{"-F", sshconfig, "-T", "-x", hostname, "uname -n"}, Env: sshenv}
			}
			{
				ret, stdout, _, _ := ssha.Run()
				if ret {
					sshhost := strings.Split(stdout, "\n")
					if realhost != sshhost[0] {
						if console {
							jsonLog.Error().Str("app", "rr").Str("id", id).Str("hostname", realhost).Msg("Hostname does not match remote host")
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
						jsonLog.Error().Str("app", "rr").Str("id", id).Str("host", realhost).Msg("Host does not exist or unreachable")
						log.Printf("%s does not exist or unreachable.", realhost)
					}
					os.Exit(1)
				}
			}
			if console {
				done()
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
				if console {
					jsonLog.Debug().Str("app", "rr").Str("id", id).Str("directory", d).Msg("copying")
				}
				log.Printf("Copying %s to %s…", d, realhost)
				var ret bool
				var stdout string
				var stderr string
				var goerr string
				var done func()
				if console {
					done = showSpinnerWhile(0)
				}
				if !sudo {
					ret, stdout, stderr, goerr = quickcopy(d, hostname, interp, sshconfig)
				} else {
					ret, stdout, stderr, goerr = sudocopy(d, hostname, id, interp, sshconfig, sudoPassword)
				}
				b64so := base64.StdEncoding.EncodeToString([]byte(stdout))
				b64se := base64.StdEncoding.EncodeToString([]byte(stderr))
				if console {
					done()
				}
				if step := "copy"; !ret {
					if !console {
						serrLog.Error().Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(step)
					} else {
						jsonLog.Error().Str("app", "rr").Str("id", id).Str("stdout", b64so).Str("stderr", b64se).Str("error", goerr).Msg(step)
						ho, bo, fo := output(stdout, hostname, STDOUT)
						he, be, fe := output(stderr, hostname, STDERR)
						hd, bd, fd := output(goerr, hostname, STDDBG)
						log.Printf("Error encountered.\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
						log.Printf("Failure copying files!")
					}
					os.Exit(1)
				} else {
					if console {
						jsonLog.Debug().Str("app", "rr").Str("id", id).Str("stdout", b64so).Str("stderr", b64se).Str("error", goerr).Msg(step)
						jsonLog.Info().Str("app", "rr").Str("id", id).Str("result", "success").Msg(step)
					}
					log.Printf("Successfully copied files")
				}
			}
		}
		log.Printf("Running %s…", script)
		if console {
			jsonLog.Debug().Str("app", "rr").Str("id", id).Str("script", script).Msg("running")
		}
		var ret bool
		var stdout string
		var stderr string
		var goerr string
		var done func()
		if console {
			done = showSpinnerWhile(1)
		}
		ret, stdout, stderr, goerr = sshexec(sudo, modscript, hostname, id, interp, sshconfig, sudoPassword)
		if console {
			done()
		}
		ho, bo, fo := output(stdout, hostname, STDOUT)
		he, be, fe := output(stderr, hostname, STDERR)
		hd, bd, fd := output(goerr, hostname, STDDBG)
		b64so := base64.StdEncoding.EncodeToString([]byte(stdout))
		b64se := base64.StdEncoding.EncodeToString([]byte(stderr))
		b64sc := base64.StdEncoding.EncodeToString([]byte(code))
		if !ret {
			failed = true
			if !console {
				serrLog.Error().Str("stdout", stdout).Str("stderr", stderr).Str("error", goerr).Msg(op)
			} else {
				jsonLog.Error().Str("app", "rr").Str("id", id).Str("code", b64sc).Str("stdout", b64so).Str("stderr", b64se).Str("error", goerr).Msg(op)
				log.Printf("Failure running script!\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
			}
		} else {
			if console {
				jsonLog.Debug().Str("app", "rr").Str("id", id).Str("code", b64sc).Str("stdout", b64so).Str("stderr", b64se).Str("error", goerr).Msg(op)
				jsonLog.Info().Str("app", "rr").Str("id", id).Str("result", "success").Msg(op)
			}
			if stdout != "" || stderr != "" || goerr != "" {
				log.Printf("Done. Output:\n%s%s%s%s%s%s", ho, bo, fo, he, be, fe)
			}
		}
	}
	if tm := fmt.Sprintf("%s", time.Since(start)); !failed {
		if console {
			jsonLog.Debug().Str("app", "rr").Str("id", id).Str("elapsed", tm).Msg("success")
		}
		log.Printf("Total run time: %s. All OK.", time.Since(start))
		os.Exit(0)
	} else {
		if console {
			jsonLog.Debug().Str("app", "rr").Str("id", id).Str("elapsed", tm).Msg("failed")
			log.Printf("Total run time: %s. Something went wrong.", time.Since(start))
		} else {
			serrLog.Debug().Str("elapsed", tm).Msg("failed")
		}
		os.Exit(1)
	}
}
