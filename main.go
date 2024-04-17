package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

	isatty "github.com/mattn/go-isatty"
	tablewriter "github.com/olekukonko/tablewriter"
	zerolog "github.com/rs/zerolog"
	lib "github.com/tongson/gl"
	terminal "golang.org/x/term"
)

var start = time.Now()

const versionNumber = "1.0.4"
const codeName = "\"Revocable Marsh\""

// constants
const cOP = "TASK"
const cRUN = "script"
const cLOG = "rr.json"
const cDOC = "README"
const cINTERP = "shell"
const cHOSTS = "rr.hosts"
const cREPAIRED = "+++++repaired+++++"
const cTIME = "02 Jan 06 15:04"

const cSTDOUT = " ┌─ stdout"
const cSTDERR = " ┌─ stderr"
const cSTDDBG = " ┌─ debug"
const cFOOTER = " └─"

type logWriter struct {
}

type optT struct {
	sudo     bool
	nopasswd bool
	teleport bool
	hostname string
	id       string
	interp   string
	config   string
	password string
}

// https://gist.github.com/jlinoff/e8e26b4ffa38d379c7f1891fd174a6d0
func getPassword(prompt string) (string, error) {
	var err error
	// Get the initial state of the terminal.
	initialTermState, err := terminal.GetState(syscall.Stdin)
	if err != nil {
		return "", err
	}

	// Restore it in the event of an interrupt.
	// CITATION: Konstantin Shaposhnikov - https://groups.google.com/forum/#!topic/golang-nuts/kTVAbtee9UA
	// syscall.SIGTERM according to staticcheck
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
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
		return "", err
	}

	// Stop looking for ^C on the channel.
	signal.Stop(c)

	// Return the password as a string.
	return string(p), err
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	fmt.Printf("\n\033[1A\033[K\033[38;2;85;85;85m%s\033[0m", time.Now().Format(time.RFC1123Z))
	return fmt.Print(" " + string(bytes))
}

func conOutput(o string, h string, c string) (string, string, string) {
	rh := ""
	rb := ""
	rf := ""
	if o != "" {
		rh = fmt.Sprintf(" %s%s\n", h, c)
		rb = fmt.Sprintf("%s\n", lib.PipeStr(h, o))
		rf = fmt.Sprintf(" %s%s\n", h, cFOOTER)
	}
	return rh, rb, rf
}

func stdWriter(stdout string, stderr string, goerr string) {
	we := bufio.NewWriter(os.Stderr)
	wo := bufio.NewWriter(os.Stdout)
	defer we.Flush()
	defer wo.Flush()
	if goerr != "" {
		_, err := we.WriteString(goerr)
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, "Something's wrong. Unable to write to STDERR at that time.")
			os.Exit(255)
		}
		err = we.Flush()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, "Something's wrong. Unable to flush writes to STDERR at that time.")
			os.Exit(255)
		}
	} else {
		_, err := we.WriteString(stderr)
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, "Something's wrong. Unable to write to STDERR at that time.")
			os.Exit(255)
		}
		err = we.Flush()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, "Something's wrong. Unable to flush writes to STDERR at that time.")
			os.Exit(255)
		}
		_, err = wo.WriteString(stdout)
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, "Something's wrong. Unable to write to STDOUT at that time.")
			os.Exit(255)
		}
		err = wo.Flush()
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, "Something's wrong. Unable to flush writes to STDOUT at that time.")
			os.Exit(255)
		}
	}
}

func sshExec(o *optT, script string) (bool, lib.RunOut) {
	tmps := fmt.Sprintf("./.__rr.scr.%s", (*o).id)
	sshenv := []string{"LC_ALL=C"}
	var ssha lib.RunArgs
	var sshb lib.RunArgs
	var sshc lib.RunArgs
	if (*o).config == "" || (*o).teleport {
		if !(*o).teleport {
			args := []string{
				"-a",
				"-T",
				"-x",
				(*o).hostname,
				fmt.Sprintf("cat - > %s", tmps),
			}
			ssha = lib.RunArgs{
				Exe:   "ssh",
				Args:  args,
				Env:   sshenv,
				Stdin: []byte(script),
			}
		} else {
			args := []string{"ssh", (*o).hostname, fmt.Sprintf("cat - > %s", tmps)}
			ssha = lib.RunArgs{Exe: "tsh", Args: args, Env: sshenv, Stdin: []byte(script)}
		}
	} else {
		args := []string{
			"-F",
			(*o).config,
			"-a",
			"-T",
			"-x",
			(*o).hostname,
			fmt.Sprintf("cat - > %s", tmps),
		}
		ssha = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte(script)}
	}
	log.Printf("CONNECTION: copying script…")
	if ret, out := ssha.Run(); !ret {
		return ret, out
	}
	var ret bool
	var out lib.RunOut
	if (*o).config == "" || (*o).teleport {
		if !(*o).sudo {
			if !(*o).teleport {
				args := []string{"-a", "-T", "-x", (*o).hostname, (*o).interp, tmps}
				sshb = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv}
			} else {
				args := []string{"ssh", (*o).hostname, (*o).interp, tmps}
				sshb = lib.RunArgs{Exe: "tsh", Args: args, Env: sshenv}
			}
		} else {
			if !(*o).teleport {
				args := []string{
					"-a",
					"-T",
					"-x",
					(*o).hostname,
					"sudo",
					"-k",
					"--prompt=\"\"",
					"-S",
					"-s",
					"--",
					(*o).interp, tmps,
				}
				sshb = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte((*o).password)}
			} else {
				args := []string{
					"ssh",
					(*o).hostname,
					"sudo",
					"-k",
					"--prompt=\"\"",
					"-S",
					"-s",
					"--",
					(*o).interp, tmps,
				}
				sshb = lib.RunArgs{Exe: "tsh", Args: args, Env: sshenv, Stdin: []byte((*o).password)}
			}
		}
	} else {
		if !(*o).sudo {
			args := []string{"-F", (*o).config, "-a", "-T", "-x", (*o).hostname, (*o).interp, tmps}
			sshb = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte((*o).password)}
		} else {
			args := []string{
				"-F",
				(*o).config,
				"-a",
				"-T",
				"-x",
				(*o).hostname,
				"sudo",
				"-k",
				"--prompt=\"\"",
				"-S",
				"-s",
				"--",
				(*o).interp,
				tmps,
			}
			sshb = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte((*o).password)}
		}
	}
	log.Printf("CONNECTION: running script…")
	ret, out = sshb.Run()
	if !ret {
		return ret, out
	}
	if (*o).config == "" || (*o).teleport {
		if !(*o).teleport {
			args := []string{
				"-a",
				"-T",
				"-x",
				(*o).hostname,
				fmt.Sprintf("rm -f %s", tmps),
			}
			sshc = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv}
		} else {
			args := []string{"ssh", (*o).hostname, fmt.Sprintf("rm -f %s", tmps)}
			sshc = lib.RunArgs{Exe: "tsh", Args: args, Env: sshenv}
		}
	} else {
		args := []string{"-F", (*o).config, "-a", "-T", "-x", (*o).hostname, fmt.Sprintf("rm -f %s", tmps)}
		sshc = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv}
	}
	log.Printf("CONNECTION: cleaning up…")
	if xret, xout := sshc.Run(); !xret {
		return xret, xout
	}
	return ret, out
}

func sudoCopy(o *optT, dir string) (bool, lib.RunOut) {
	tmpd := fmt.Sprintf(".__rr.dir.%s", (*o).id)
	tmpf := fmt.Sprintf("./.__rr.tar.%s", (*o).id)
	tarcmd := `
	set -efu
	LC_ALL=C
	unset IFS
	tar -C %s -cpf - . | tar -C / --overwrite --no-same-owner -ompxf -
	rm -rf %s
	rm -f %s
	`
	tarexec := fmt.Sprintf(tarcmd, tmpd, tmpd, tmpf)
	sshenv := []string{"LC_ALL=C"}
	var untar1 lib.RunArgs
	if (*o).config == "" || (*o).teleport {
		if !(*o).teleport {
			untar1 = lib.RunArgs{
				Exe: "ssh",
				Args: []string{
					"-a",
					"-T",
					"-x",
					(*o).hostname,
					fmt.Sprintf("cat - > %s", tmpf),
				},
				Env:   sshenv,
				Stdin: []byte(tarexec),
			}
		} else {
			untar1 = lib.RunArgs{Exe: "tsh", Args: []string{
				"ssh",
				(*o).hostname,
				fmt.Sprintf("cat - > %s", tmpf)},
				Env:   sshenv,
				Stdin: []byte(tarexec),
			}
		}
	} else {
		args := []string{"-F", (*o).config, "-a", "-T", "-x", (*o).hostname, fmt.Sprintf("cat - > %s", tmpf)}
		untar1 = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte(tarexec)}
	}
	if ret, out := untar1.Run(); !ret {
		return ret, out
	}
	untarDefault := `
	RRHOST="%s"
	RRSRC="%s"
	RRDEST="%s"
	RRSCRIPT="%s"
	ssh -T -x "$RRHOST" mkdir "$RRDEST"
	tar -C "$RRSRC" -cpzf - . | ssh -a -T -x "$RRHOST" tar -C "$RRDEST" --no-same-owner -omxpzf -
	`
	teleportDefault := `
	RRHOST="%s"
	RRSRC="%s"
	RRDEST="%s"
	RRSCRIPT="%s"
	tsh ssh "$RRHOST" mkdir "$RRDEST"
	tar -C "$RRSRC" -cpzf - . | tsh ssh "$RRHOST" tar -C "$RRDEST" --no-same-owner -omxpzf -
	`
	untarConfig := `
	RRHOST="%s"
	RRSRC="%s"
	RRDEST="%s"
	RRCONFIG="%s"
	RRSCRIPT="%s"
	ssh -F "$RRCONFIG" -T -x "$RRHOST" mkdir "$RRDEST"
	tar -C "$RRSRC" -cpzf - . | ssh -F "$RRCONFIG" -a -T -x "$RRHOST" tar -C "$RRDEST" --no-same-owner -omxpzf -
	`
	tarenv := []string{"LC_ALL=C"}
	var untar2 lib.RunArgs
	if (*o).config == "" || (*o).teleport {
		if !(*o).teleport {
			untar2 = lib.RunArgs{
				Exe: (*o).interp,
				Args: []string{
					"-c",
					fmt.Sprintf(untarDefault, (*o).hostname, dir, tmpd, tmpf),
				},
				Env: tarenv,
			}
		} else {
			untar2 = lib.RunArgs{Exe: (*o).interp, Args: []string{
				"-c",
				fmt.Sprintf(teleportDefault, (*o).hostname, dir, tmpd, tmpf)},
				Env: tarenv,
			}
		}
	} else {
		untar2 = lib.RunArgs{Exe: (*o).interp, Args: []string{
			"-c",
			fmt.Sprintf(untarConfig, (*o).hostname, dir, tmpd, (*o).config, tmpf)},
			Env: tarenv,
		}
	}
	if ret, out := untar2.Run(); !ret {
		return ret, out
	}
	var untar3 lib.RunArgs
	if (*o).config == "" || (*o).teleport {
		if !(*o).teleport {
			args := []string{
				"-a",
				"-T",
				"-x",
				(*o).hostname,
				"sudo",
				"-k",
				"--prompt=\"\"",
				"-S",
				"-s",
				"--",
				(*o).interp,
				tmpf,
			}
			untar3 = lib.RunArgs{
				Exe:   "ssh",
				Args:  args,
				Env:   sshenv,
				Stdin: []byte((*o).password),
			}
		} else {
			args := []string{"ssh", (*o).hostname, "sudo", "-k", "--prompt=\"\"", "-S", "-s", "--", (*o).interp, tmpf}
			untar3 = lib.RunArgs{Exe: "tsh", Args: args, Env: sshenv, Stdin: []byte((*o).password)}
		}
	} else {
		args := []string{"-F",
			(*o).config,
			"-a",
			"-T",
			"-x",
			(*o).hostname,
			"sudo",
			"-k",
			"--prompt=\"\"",
			"-S",
			"-s",
			"--",
			(*o).interp, tmpf,
		}
		untar3 = lib.RunArgs{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte((*o).password)}
	}
	return untar3.Run()
}

func quickCopy(o *optT, dir string) (bool, lib.RunOut) {
	untarDefault := `
	set -o errexit -o nounset -o noglob
	tar -C %s -cpzf - . | ssh -a -T -x %s tar -C / --overwrite --no-same-owner -omxpzf -
	`
	untarTeleport := `
	set -o errexit -o nounset -o noglob
	tar -C %s -cpzf - . | tsh ssh %s tar -C / --overwrite --no-same-owner -omxpzf -
	`
	untarConfig := `
	tar -C %s -cpzf - . | ssh -F %s -a -T -x %s tar -C / --overwrite --no-same-owner -omxpzf -
	`
	tarenv := []string{"LC_ALL=C"}
	var untar lib.RunArgs
	if (*o).config == "" || (*o).teleport {
		if !(*o).teleport {
			untar = lib.RunArgs{
				Exe:  (*o).interp,
				Args: []string{"-c", fmt.Sprintf(untarDefault, dir, (*o).hostname)},
				Env:  tarenv,
			}
		} else {
			untar = lib.RunArgs{Exe: (*o).interp, Args: []string{
				"-c",
				fmt.Sprintf(untarTeleport, dir, (*o).hostname)},
				Env: tarenv,
			}
		}
	} else {
		untar = lib.RunArgs{Exe: (*o).interp, Args: []string{"-c",
			fmt.Sprintf(untarConfig, dir, (*o).config, (*o).hostname)},
			Env: tarenv,
		}
	}
	return untar.Run()
}

func main() {
	runtime.MemProfileRate = 0
	defer lib.RecoverPanic()
	var opt optT
	var plain bool = false
	var console bool = false
	var failed bool = false
	var result string = "ok"
	var dump bool = false
	var report bool = false
	if call := os.Args[0]; len(call) < 3 || call[len(call)-2:] == "rr" {
		log.SetOutput(io.Discard)
	} else if call[len(call)-3:] == "rrp" {
		plain = true
		log.SetOutput(io.Discard)
	} else if call[len(call)-3:] == "rrv" {
		console = true
		log.SetOutput(new(logWriter))
	} else if call[len(call)-3:] == "rrd" {
		dump = true
		log.SetOutput(io.Discard)
	} else if call[len(call)-3:] == "rrl" {
		report = true
		log.SetOutput(io.Discard)
	} else if call[len(call)-3:] == "rrs" {
		opt.sudo = true
	} else if call[len(call)-3:] == "rrt" {
		opt.teleport = true
	} else if call[len(call)-3:] == "rro" {
		opt.sudo = true
		opt.teleport = true
	} else if call[len(call)-3:] == "rru" {
		opt.sudo = true
		opt.nopasswd = true
	} else {
		valid := `Valid modes:
	rr  = local or ssh
	rrs = ssh + sudo
	rru = ssh + sudo + nopasswd
	rrt = teleport
	rro = teleport + sudo
	rrd = dump
	rrv = forced verbose
	rrl = report`
		fmt.Fprintf(os.Stderr, "Unsupported executable name.\n%s\n", valid)
		os.Exit(1)
	}
	if report {
		hdrs := []string{
			"ID",
			"Target",
			"Started",
			"Namespace",
			"Script",
			"Task",
			"Duration",
			"Result",
		}
		var data [][]string
		rrl, err := os.Open(cLOG)
		if err != nil {
			lib.Panicf("Missing %s.", cLOG)
			os.Exit(1)
		}
		var maxSz int
		defer func() {
			err := rrl.Close()
			if err != nil {
				lib.Panic("Problem closing log.")
				os.Exit(1)
			}
		}()
		scanner := bufio.NewScanner(rrl)
		rrlInfo, err := rrl.Stat()
		if err != nil {
			lib.Panicf("Unable to open %s.", cLOG)
			os.Exit(1)
		}
		maxSz = int(rrlInfo.Size())
		buf := make([]byte, 0, maxSz)
		scanner.Buffer(buf, maxSz)
		for scanner.Scan() {
			log := make(map[string]string)
			err := json.Unmarshal(scanner.Bytes(), &log)
			if err != nil {
				lib.Panicf("Unable to decode %s.", cLOG)
				os.Exit(1)
			}
			if log["duration"] != "" {
				data = append(data, []string{log["id"],
					log["target"],
					log["start"],
					log["namespace"],
					log["script"],
					log["task"],
					log["duration"],
					log["message"]})
			}
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetRowLine(true) // Enable row line
		table.SetCenterSeparator("⋅")
		table.SetColumnSeparator("│")
		table.SetRowSeparator("─")
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetHeader(hdrs)
		table.SetFooter(hdrs)
		table.SetBorder(true)
		table.SetRowLine(true)
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
		)
		table.SetFooterColor(
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
		)
		table.AppendBulk(data)
		table.Render()
		os.Exit(0)
	}
	log.SetFlags(0)
	zerolog.TimeFieldFormat = time.RFC3339
	var serrLog zerolog.Logger
	if !dump && !plain {
		if isatty.IsTerminal(os.Stdout.Fd()) {
			console = true
			log.SetOutput(new(logWriter))
			log.Printf("rr %s %s", versionNumber, codeName)
		}
	}
	if !console {
		serrLog = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}
	isDir := lib.StatPath("directory")
	isFile := lib.StatPath("file")
	var offset int
	var hostname string
	var id string
	{
		h := new(maphash.Hash)
		uid := fmt.Sprintf("%016X", h.Sum64())
		id = string([]rune(uid)[:8])
		opt.id = id
	}
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
		opt.hostname = hostname
	} else {
		offset = 2
		hostname = os.Args[1]
		opt.hostname = hostname
	}

	// Handle readmes
	{
		isReadme := func(s string) (bool, string) {
			s = strings.TrimSuffix(s, "/")
			s = strings.TrimSuffix(s, ":")
			match, _ := lib.FileGlob(fmt.Sprintf("%s/%s*", s, cDOC))
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
			fmt.Printf("%s┐\n", line)
			if console {
				fmt.Printf(" \033[37;1m%s\033[0m │\n", pps)
			} else {
				fmt.Printf(" %s │\n", pps)
			}
			fmt.Printf("%s┘\n", line)
			if console {
				for _, each := range lib.FileLines(s) {
					fmt.Printf(" \033[38;2;85;85;85m⋮\033[0m %s\n", each)
				}
				fmt.Printf("\n")
			} else {
				fmt.Print(lib.FileRead(s))
			}
		}
		if found1, readme1 := isReadme(os.Args[1]); found1 && readme1 != "" {
			log.Print("Showing README…")
			printReadme(readme1)
			os.Exit(0)
		} else if len(os.Args) > 2 {
			if found2, readme2 := isReadme(os.Args[2]); found2 && readme2 != "" {
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
	var interp string
	jsonFile, _ := os.OpenFile(cLOG, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	jsonLog := zerolog.New(jsonFile).With().Timestamp().Logger()
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
				serrLog.Fatal().Str("namespace", namespace).Msg("Namespace is not a directory")
				os.Exit(1)
			}
		}
		if !isDir(fmt.Sprintf("%s/%s", namespace, script)) {
			if console {
				lib.Panicf("`%s/%s` is not a diretory.", namespace, script)
			} else {
				serrLog.Fatal().
					Str("namespace", namespace).
					Str("script", script).
					Msg("namespace/script is not a directory")
				os.Exit(1)
			}
		}
		if !isFile(fmt.Sprintf("%s/%s/%s", namespace, script, cRUN)) {
			if console {
				lib.Panicf(
					"`%s/%s/%s` actual script not found.",
					namespace,
					script,
					cRUN,
				)
			} else {
				serrLog.Fatal().
					Str("namespace", namespace).
					Str("script", script).
					Msg("Actual script is missing")
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
		fnWalkDir:= lib.PathWalker(&sh)
		if isDir(".lib") {
			lib.Assert(filepath.WalkDir(".lib", fnWalkDir), "filepath.WalkDir(\".lib\")")
		}

		if isDir(namespace + "/.lib") {
			lib.Assert(
				filepath.WalkDir(namespace+"/.lib", fnWalkDir),
				"filepath.WalkDir(namespace+\".lib\")",
			)
		}
		if isDir(namespace + "/" + script + "/.lib") {
			lib.Assert(
				filepath.WalkDir(namespace+"/"+script+"/.lib", fnWalkDir),
				"filepath.WalkDir(namespace+\".lib\")",
			)
		}
		if opt.sudo {
			if !opt.nopasswd {
				str, err := getPassword("sudo password: ")
				if err != nil {
					if console {
						lib.Panicf(
							"`%s/%s/%s` actual script not found.",
							namespace,
							script,
							cRUN,
						)
					} else {
						serrLog.Fatal().
							Str("namespace", namespace).
							Str("script", script).
							Msg("Actual script is missing")
						os.Exit(1)
					}
				}
				opt.password = fmt.Sprintf("%s\n", str)
			}
		}
		//Pass environment variables with `rr` prefix
		for _, e := range os.Environ() {
			if strings.HasPrefix(e, "rr__") {
				sh.WriteString("export " + strings.TrimPrefix(e, "rr__") + "\n")
			}
		}
		if len(arguments) > 0 {
			arguments = lib.InsertStr(arguments, "set --", 0)
			sh.WriteString(strings.Join(arguments, " "))
		}
		code = lib.FileRead(namespace + "/" + script + "/" + cRUN)
		sh.WriteString("\n" + code)
	}
	modscript := sh.String()
	if dump {
		fmt.Print(modscript)
		os.Exit(0)
	}
	interp = lib.FileRead(fmt.Sprintf("%s/%s/%s", namespace, script, cINTERP))
	interp = strings.TrimSuffix(interp, "\n")
	if interp == "" {
		opt.interp = "sh"
		interp = "sh"
	} else {
		opt.interp = interp
	}
	var op string
	{
		var ok bool
		op, ok = os.LookupEnv(cOP)
		if !ok {
			op = lib.FileRead(cOP)
			op = strings.Split(op, "\n")[0]
			if op == "" {
				op = "UNDEFINED"
			}
		}
	}
	jsonLog.Info().
		Str("app", "rr").
		Str("id", id).
		Str("namespace", namespace).
		Str("script", script).
		Str("target", hostname).
		Msg(op)
	log.Printf("Running %s:%s via %s…", namespace, script, hostname)
	if hostname == "local" || hostname == "localhost" {
		if opt.sudo {
			log.Printf("Invoked sudo+ssh mode via local, ignored mode, just `sudo rr`.")
		}
		untar := `
                LC_ALL=C
                set -o errexit -o nounset -o noglob
                unset IFS
                tar -C %s -cpf - . | tar -C / --no-same-owner -ompxf -
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
				jsonLog.Debug().
					Str("app", "rr").
					Str("id", id).
					Str("directory", d).
					Msg("copying")
				rargs := lib.RunArgs{
					Exe:  interp,
					Args: []string{"-c", fmt.Sprintf(untar, d)},
				}
				ret, out := rargs.Run()
				b64so := base64.StdEncoding.EncodeToString([]byte(out.Stdout))
				b64se := base64.StdEncoding.EncodeToString([]byte(out.Stderr))
				if step := "copy"; !ret {
					jsonLog.Error().
						Str("app", "rr").
						Str("id", id).
						Str("stdout", b64so).
						Str("stderr", b64se).
						Str("error", out.Error).
						Msg(step)
					if plain {
						stdWriter(out.Stdout, out.Stderr, out.Error)
					} else if !console {
						serrLog.Error().
							Str("stdout", out.Stdout).
							Str("stderr", out.Stderr).
							Str("error", out.Error).
							Msg(step)
					} else {
						ho, bo, fo := conOutput(out.Stdout, hostname, cSTDOUT)
						he, be, fe := conOutput(out.Stderr, hostname, cSTDERR)
						hd, bd, fd := conOutput(out.Error, hostname, cSTDDBG)
						log.Printf("Error encountered.\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
						log.Printf("Failure copying files!")
					}
					os.Exit(1)
				} else {
					jsonLog.Debug().
						Str("app", "rr").
						Str("id", id).
						Str("stdout", b64so).
						Str("stderr", b64se).
						Str("error", out.Error).
						Msg(step)
					jsonLog.Info().Str("app", "rr").Str("id", id).Str("result", "success").Msg(step)
					if !plain {
						log.Printf("Successfully copied files")
					}
				}
			}
		}
		if op == "UNDEFINED" {
			log.Printf("Running %s…", script)
			jsonLog.Debug().
				Str("app", "rr").
				Str("id", id).
				Str("script", script).
				Msg("running")
		} else {
			msgop := strings.TrimSuffix(op, "\n")
			log.Printf("%s…", msgop)
			jsonLog.Debug().Str("app", "rr").Str("id", id).Str("script", script).Msg(msgop)
		}
		rargs := lib.RunArgs{Exe: interp, Stdin: []byte(modscript)}
		ret, out := rargs.Run()
		ho, bo, fo := conOutput(out.Stdout, hostname, cSTDOUT)
		he, be, fe := conOutput(out.Stderr, hostname, cSTDERR)
		hd, bd, fd := conOutput(out.Error, hostname, cSTDDBG)
		b64so := base64.StdEncoding.EncodeToString([]byte(out.Stdout))
		b64se := base64.StdEncoding.EncodeToString([]byte(out.Stderr))
		b64sc := base64.StdEncoding.EncodeToString([]byte(code))
		if !ret {
			failed = true
			jsonLog.Error().
				Str("app", "rr").
				Str("id", id).
				Str("code", b64sc).
				Str("stdout", b64so).
				Str("stderr", b64se).
				Str("error", out.Error).
				Msg(op)
			if plain {
				stdWriter(out.Stdout, out.Stderr, out.Error)
			} else if !console {
				serrLog.Error().
					Str("stdout", out.Stdout).
					Str("stderr", out.Stderr).
					Str("error", out.Error).
					Msg(op)
			} else {
				log.Printf("Failure running script!\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
			}
		} else {
			scanner := bufio.NewScanner(strings.NewReader(out.Stdout))
			for scanner.Scan() {
				if scanner.Text() == cREPAIRED {
					result = "repaired"
				}
			}
			jsonLog.Debug().
				Str("app", "rr").
				Str("id", id).
				Str("code", b64sc).
				Str("stdout", b64so).
				Str("stderr", b64se).
				Str("error", out.Error).
				Msg(op)
			jsonLog.Info().Str("app", "rr").Str("id", id).Str("result", result).Msg(op)
			if plain {
				stdWriter(out.Stdout, out.Stderr, out.Error)
			} else if out.Stdout != "" || out.Stderr != "" || out.Error != "" {
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
				jsonLog.Debug().Str("app", "rr").Str("id", id).Str("directory", d).Msg("copying")
				tarenv := []string{"LC_ALL=C"}
				untar := `
				tar -C %s -cf - . | tar -C %s --no-same-owner --overwrite -omxpf -
				`
				rsargs := lib.RunArgs{Exe: interp, Args: []string{"-c", fmt.Sprintf(untar, d, destination)}, Env: tarenv}
				ret, out := rsargs.Run()
				b64so := base64.StdEncoding.EncodeToString([]byte(out.Stdout))
				b64se := base64.StdEncoding.EncodeToString([]byte(out.Stderr))
				if step := "copy"; !ret {
					jsonLog.Error().
						Str("app", "rr").
						Str("id", id).
						Str("stdout", b64so).
						Str("stderr", b64se).
						Str("error", out.Error).
						Msg(step)
					if plain {
						stdWriter(out.Stdout, out.Stderr, out.Error)
					} else if !console {
						serrLog.Error().Str("stdout", out.Stdout).Str("stderr", out.Stderr).Str("error", out.Error).Msg(step)
					} else {
						ho, bo, fo := conOutput(out.Stdout, hostname, cSTDOUT)
						he, be, fe := conOutput(out.Stderr, hostname, cSTDERR)
						hd, bd, fd := conOutput(out.Error, hostname, cSTDDBG)
						log.Printf("Error encountered.\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
						log.Printf("Failure copying files!")
					}
					os.Exit(1)
				} else {
					jsonLog.Debug().
						Str("app", "rr").
						Str("id", id).
						Str("stdout", b64so).
						Str("stderr", b64se).
						Str("error", out.Error).
						Msg(step)
					jsonLog.Info().Str("app", "rr").Str("id", id).Str("result", "success").Msg(step)
					if !plain {
						log.Printf("Successfully copied files")
					}
				}
			}
		}
		log.Printf("Running %s…", script)
		jsonLog.Debug().Str("app", "rr").Str("id", id).Str("script", script).Msg("running")
		nsargs := lib.RunArgs{Exe: "nsenter", Args: []string{"-a", "-r", "-t", hostname, interp, "-c", modscript}}
		ret, out := nsargs.Run()
		ho, bo, fo := conOutput(out.Stdout, hostname, cSTDOUT)
		he, be, fe := conOutput(out.Stderr, hostname, cSTDERR)
		hd, bd, fd := conOutput(out.Error, hostname, cSTDDBG)
		b64so := base64.StdEncoding.EncodeToString([]byte(out.Stdout))
		b64se := base64.StdEncoding.EncodeToString([]byte(out.Stderr))
		b64sc := base64.StdEncoding.EncodeToString([]byte(code))
		if !ret {
			failed = true
			jsonLog.Error().
				Str("app", "rr").
				Str("id", id).
				Str("code", b64sc).
				Str("stdout", b64so).
				Str("stderr", b64se).
				Str("error", out.Error).
				Msg(op)
			if plain {
				stdWriter(out.Stdout, out.Stderr, out.Error)
			} else if !console {
				serrLog.Error().Str("stdout", out.Stdout).Str("stderr", out.Stderr).Str("error", out.Error).Msg(op)
			} else {
				log.Printf("Failure running script!\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
			}
		} else {
			scanner := bufio.NewScanner(strings.NewReader(out.Stdout))
			for scanner.Scan() {
				if scanner.Text() == cREPAIRED {
					result = "repaired"
				}
			}
			jsonLog.Debug().
				Str("app", "rr").
				Str("id", id).
				Str("code", b64sc).
				Str("stdout", b64so).
				Str("stderr", b64se).
				Str("error", out.Error).
				Msg(op)
			jsonLog.Info().Str("app", "rr").Str("id", id).Str("result", result).Msg(op)
			if plain {
				stdWriter(out.Stdout, out.Stderr, out.Error)
			} else if out.Stdout != "" || out.Stderr != "" || out.Error != "" {
				log.Printf("Done. Output:\n%s%s%s%s%s%s", ho, bo, fo, he, be, fe)
			}
		}
	} else {
		if lib.IsFile(cHOSTS) && !opt.teleport {
			opt.config = cHOSTS
		}
		var realhost string
		if rh := strings.Split(hostname, "@"); len(rh) == 1 {
			realhost = hostname
		} else {
			realhost = rh[1]
		}
		{
			sshenv := []string{"LC_ALL=C"}
			var ssha lib.RunArgs
			if opt.config == "" || opt.teleport {
				if !opt.teleport {
					ssha = lib.RunArgs{Exe: "ssh", Args: []string{"-T", "-x", opt.hostname, "uname -n"}, Env: sshenv}
				} else {
					ssha = lib.RunArgs{Exe: "tsh", Args: []string{"ssh", opt.hostname, "uname -n"}, Env: sshenv}
				}
			} else {
				ssha = lib.RunArgs{Exe: "ssh", Args: []string{
					"-F",
					opt.config,
					"-T",
					"-x",
					opt.hostname,
					"uname -n",
				}, Env: sshenv}
			}
			{
				log.Printf("CONNECTION: checking for hostname match…")
				ret, out := ssha.Run()
				if ret {
					sshhost := strings.Split(out.Stdout, "\n")
					if realhost != sshhost[0] {
						jsonLog.Error().
							Str("app", "rr").
							Str("id", id).
							Str("hostname", realhost).
							Msg("Hostname does not match remote host")
						if plain {
							stdWriter("", "Hostname does not match remote host.", "")
						} else if console {
							log.Printf("Hostname %s does not match remote host.", realhost)
						} else {
							serrLog.Error().Str("hostname", realhost).Msg("Hostname does not match remote host")
						}
						os.Exit(1)
					} else {
						if !plain {
							log.Printf("Remote host is %s\n", sshhost[0])
						}
					}
				} else {
					hostErr := fmt.Sprintf("Host does not exist or unreachable. [%s]", out.Stderr)
					jsonLog.Error().
						Str("app", "rr").
						Str("id", id).
						Str("host", realhost).
						Msg(hostErr)
					if plain {
						stdWriter("", hostErr, "")
					} else if !console {
						serrLog.Error().Str("host", realhost).Msg(hostErr)
					} else {
						log.Printf("%s does not exist or unreachable. [%s]", realhost, out.Stderr)
					}
					os.Exit(1)
				}
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
				jsonLog.Debug().Str("app", "rr").Str("id", id).Str("directory", d).Msg("copying")
				log.Printf("CONNECTION: copying %s to %s…", d, realhost)
				var ret bool
				var out lib.RunOut
				if !opt.sudo {
					ret, out = quickCopy(&opt, d)
				} else {
					ret, out = sudoCopy(&opt, d)
				}
				b64so := base64.StdEncoding.EncodeToString([]byte(out.Stdout))
				b64se := base64.StdEncoding.EncodeToString([]byte(out.Stderr))
				if step := "copy"; !ret {
					jsonLog.Error().
						Str("app", "rr").
						Str("id", id).
						Str("stdout", b64so).
						Str("stderr", b64se).
						Str("error", out.Error).
						Msg(step)
					if plain {
						stdWriter(out.Stdout, out.Stderr, out.Error)
					} else if !console {
						serrLog.Error().Str("stdout", out.Stdout).Str("stderr", out.Stderr).Str("error", out.Error).Msg(step)
					} else {
						ho, bo, fo := conOutput(out.Stdout, hostname, cSTDOUT)
						he, be, fe := conOutput(out.Stderr, hostname, cSTDERR)
						hd, bd, fd := conOutput(out.Error, hostname, cSTDDBG)
						log.Printf("Error encountered.\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
						log.Printf("Failure copying files!")
					}
					os.Exit(1)
				} else {
					jsonLog.Debug().
						Str("app", "rr").
						Str("id", id).
						Str("stdout", b64so).
						Str("stderr", b64se).
						Str("error", out.Error).
						Msg(step)
					jsonLog.Info().Str("app", "rr").Str("id", id).Str("result", "success").Msg(step)
					if !plain {
						log.Printf("Successfully copied files")
					}
				}
			}
		}
		if !plain {
			log.Printf("Running %s…", script)
		}
		jsonLog.Debug().Str("app", "rr").Str("id", id).Str("script", script).Msg("running")
		var ret bool
		var out lib.RunOut
		ret, out = sshExec(&opt, modscript)
		ho, bo, fo := conOutput(out.Stdout, hostname, cSTDOUT)
		he, be, fe := conOutput(out.Stderr, hostname, cSTDERR)
		hd, bd, fd := conOutput(out.Error, hostname, cSTDDBG)
		b64so := base64.StdEncoding.EncodeToString([]byte(out.Stdout))
		b64se := base64.StdEncoding.EncodeToString([]byte(out.Stderr))
		b64sc := base64.StdEncoding.EncodeToString([]byte(code))
		if !ret {
			failed = true
			jsonLog.Error().
				Str("app", "rr").
				Str("id", id).
				Str("code", b64sc).
				Str("stdout", b64so).
				Str("stderr", b64se).
				Str("error", out.Error).
				Msg(op)
			if plain {
				stdWriter(out.Stdout, out.Stderr, out.Error)
			} else if !console {
				serrLog.Error().Str("stdout", out.Stdout).Str("stderr", out.Stderr).Str("error", out.Error).Msg(op)
			} else {
				log.Printf("Failure running script!\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
			}
		} else {
			scanner := bufio.NewScanner(strings.NewReader(out.Stdout))
			for scanner.Scan() {
				if scanner.Text() == cREPAIRED {
					result = "repaired"
				}
			}
			jsonLog.Debug().
				Str("app", "rr").
				Str("id", id).
				Str("code", b64sc).
				Str("stdout", b64so).
				Str("stderr", b64se).
				Str("error", out.Error).
				Msg(op)
			jsonLog.Info().Str("app", "rr").Str("id", id).Str("result", result).Msg(op)
			if plain {
				stdWriter(out.Stdout, out.Stderr, out.Error)
			} else if out.Stdout != "" || out.Stderr != "" || out.Error != "" {
				log.Printf("Done. Output:\n%s%s%s%s%s%s", ho, bo, fo, he, be, fe)
			}
		}
	}
	{
		tm := time.Since(start).Truncate(time.Second).String()
		if tm == "0s" {
			tm = "<1s"
		}
		if !failed {
			jsonLog.Debug().
				Str("app", "rr").
				Str("id", id).
				Str("start", start.Format(cTIME)).
				Str("task", op).
				Str("target", hostname).
				Str("namespace", namespace).
				Str("script", script).
				Str("duration", tm).
				Msg(result)
			if !plain {
				log.Printf("Total run time: %s. All OK.", tm)
			}
			os.Exit(0)
		} else {
			jsonLog.Debug().
				Str("app", "rr").
				Str("id", id).
				Str("start", start.Format(cTIME)).
				Str("task", op).
				Str("target", hostname).
				Str("namespace", namespace).
				Str("script", script).
				Str("duration", tm).
				Msg("failed")
			if console && !plain {
				log.Printf("Total run time: %s. Something went wrong.", tm)
			} else {
				serrLog.Debug().Str("duration", tm).Msg("failed")
			}
			os.Exit(1)
		}
	}
}
