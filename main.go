package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"hash/maphash"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	isatty "github.com/mattn/go-isatty"
	lib "github.com/tongson/gl"
	terminal "golang.org/x/term"
)

var start = time.Now()
var serrLog *slog.Logger

type logWriter struct{}

type optT struct {
	hostname  string
	id        string
	interp    string
	password  string
	sshconfig string
	mode      int
	call      int
	sudo      int
	sudopwd   int
}

type scriptT struct {
	namespace string
	script    string
	nsscript  string
	prelude   string
	epilogue  string
	lib       string
	code      string
	log       string
	interp    string
}

type b64String struct {
	stdout string
	stderr string
	code   string
}

func logInt() {
	tm := time.Since(start).Truncate(time.Second).String()
	if tm == "0s" {
		tm = "<1s"
	}
	jsonFile, _ := os.OpenFile(cLOG, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	defer jsonFile.Close()
	jsonLog := slog.New(slog.NewJSONHandler(jsonFile, &slog.HandlerOptions{Level: slog.LevelDebug}))
	jsonLog.Debug("sigint", "app", "rr", "id", "???", "start", start.Format(cTIME), "task", "INTERRUPTED", "target", "???", "namespace", "???", "script", "???", "duration", tm)
	_ = jsonFile.Close()
}

// CITATION: Konstantin Shaposhnikov - https://groups.google.com/forum/#!topic/golang-nuts/kTVAbtee9UA
// REFERENCE: https://gist.github.com/jlinoff/e8e26b4ffa38d379c7f1891fd174a6d0
var initTermState *terminal.State
var cleanUpFn func(string) = func(a string) {
	_, _ = fmt.Fprintf(os.Stderr, "%s.", a)
}

func init() {
	var err error
	initTermState, err = terminal.GetState(int(os.Stdin.Fd()))
	if err != nil {
		return
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logInt()
		cleanUpFn("Caught signal. Exiting.\n")
		_ = terminal.Restore(int(os.Stdin.Fd()), initTermState)
		os.Exit(2)
	}()
}

func setupScript(o optT, offset int) scriptT {
	var sh strings.Builder
	var namespace string
	var script string
	var prelude string
	var epilogue string
	var dumplib string
	var code string
	var oplog string

	var s []string
	// Old behavior. Allowed hacky tab completion by replacing the '/' with ':'.
	// Ditched because of the README feature.
	// s := strings.Split(os.Args[offset], "/")
	if len(s) < 2 {
		s = strings.Split(os.Args[offset], ":")
	}
	namespace, script = s[0], s[1]
	var interp string = lib.FileRead(fmt.Sprintf("%s/%s/%s", namespace, script, cINTERP))
	interp = strings.TrimSuffix(interp, "\n")
	if interp == "" {
		interp = "sh"
	}

	if !lib.IsDir(namespace) {
		switch o.mode {
		case cTerm, cPlain:
			_, _ = fmt.Fprintf(os.Stderr, "Namespace `%s` is not a directory\n", namespace)
			os.Exit(2)
		case cJson:
			serrLog.Error("Namespace is not a directory", "namespace", namespace)
			os.Exit(2)
		}
	}
	if !lib.IsDir(fmt.Sprintf("%s/%s", namespace, script)) {
		switch o.mode {
		case cTerm, cPlain:
			_, _ = fmt.Fprintf(os.Stderr, "`%s/%s` is not a directory\n", namespace, script)
			os.Exit(2)
		case cJson:
			serrLog.Error("namespace/script is not a directory", "namespace", namespace, "script", script)
			os.Exit(2)
		}
	}
	if !lib.IsFile(fmt.Sprintf("%s/%s/%s", namespace, script, cRUN)) {
		switch o.mode {
		case cTerm, cPlain:
			_, _ = fmt.Fprintf(os.Stderr, "`%s/%s/%s` script not found\n", namespace, script, cRUN)
			os.Exit(2)
		case cJson:
			serrLog.Error("Script not found", "namespace", namespace, "script", script)
			os.Exit(2)
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

	// Set LOG field
	if eop, ok := os.LookupEnv(cOP); !ok {
		if len(arguments) == 0 {
			oplog = "UNDEFINED"
		} else {
			oplog = strings.Join(arguments, " ")
		}
	} else {
		oplog = eop
	}
	fnWalkDir := lib.PathWalker(&sh)
	if lib.IsDir(".lib") {
		if err := filepath.WalkDir(".lib", fnWalkDir); err != nil {
			_, _ = fmt.Fprint(os.Stderr, "Problem accessing .lib")
			os.Exit(255)
		}
	}
	if lib.IsDir(namespace + "/.lib") {
		if err := filepath.WalkDir(namespace+"/.lib", fnWalkDir); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Problem accessing %s/.lib\n", namespace)
			os.Exit(255)
		}
	}
	if lib.IsDir(namespace + "/" + script + "/.lib") {
		if err := filepath.WalkDir(namespace+"/"+script+"/.lib", fnWalkDir); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Problem accessing %s/%s/.lib\n", namespace, script)
			os.Exit(255)
		}
	}
	dumplib = sh.String()
	//Pass environment variables with `rr` prefix
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "rr__") {
			sh.WriteString("export " + strings.TrimPrefix(e, "rr__") + "\n")
		}
	}
	if len(arguments) > 0 {
		arguments = lib.InsertStr(arguments, "set --", 0)
		sh.WriteString(strings.Join(arguments, " "))
		sh.WriteString("\n")
	}
	if lib.IsFile(namespace + "/" + script + "/" + cPRE) {
		if c := lib.FileRead(namespace + "/" + script + "/" + cPRE); lib.IsFile(cINC) {
			inc := lib.FileRead(cINC) + "\n"
			prelude = sh.String() + inc + c
		} else {
			prelude = sh.String() + c
		}
	}
	if lib.IsFile(namespace + "/" + script + "/" + cPOST) {
		if c := lib.FileRead(namespace + "/" + script + "/" + cPOST); lib.IsFile(cINC) {
			inc := lib.FileRead(cINC) + "\n"
			epilogue = sh.String() + inc + c
		} else {
			epilogue = sh.String() + c
		}
	}
	if c := lib.FileRead(namespace + "/" + script + "/" + cRUN); lib.IsFile(cINC) {
		inc := lib.FileRead(cINC) + "\n"
		code = inc + c
	} else {
		code = c
	}
	sh.WriteString(code)
	return scriptT{
		nsscript:  sh.String(),
		namespace: namespace,
		script:    script,
		code:      code,
		lib:       dumplib,
		prelude:   prelude,
		epilogue:  epilogue,
		log:       oplog,
		interp:    interp,
	}
}

func b64(stdout string, stderr string, code string) b64String {
	var s b64String
	s.stdout = base64.StdEncoding.EncodeToString([]byte(stdout))
	s.stderr = base64.StdEncoding.EncodeToString([]byte(stderr))
	s.code = base64.StdEncoding.EncodeToString([]byte(code))
	return s
}

func since(s time.Time) string {
	tm := time.Since(s).Truncate(time.Second).String()
	if tm == "0s" {
		tm = "<1s"
	}
	return tm
}

func getPassword(prompt string) (string, error) {
	var err error
	fmt.Print(prompt)
	p, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println("")
	if err != nil {
		return "", err
	}
	return string(p), err
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Printf(cANSI, time.Now().Format(time.RFC822Z), string(bytes))
}

func soOutput(h string, m int) func(string) {
	switch m {
	case cTerm:
		return func(so string) {
			if strings.Contains(so, "\n") {
				fmt.Printf(" %s │ %s", h, so)
			} else if so != "" {
				fmt.Printf(" %s │ %s\n", h, so)
			}
		}
	default:
		return func(_ string) {
		}
	}
}

func conOutput(o string, h string, c string) (string, string, string) {
	rh, rb, rf := "", "", ""
	if o != "" {
		rh = fmt.Sprintf(" %s%s\n", h, c)
		rb = fmt.Sprintf("%s\n", lib.PipeStr(h, o))
		rf = fmt.Sprintf(" %s%s\n", h, cFOOTER)
	}
	return rh, rb, rf
}

func stdWriter(stdout string, stderr string) {
	if stdout != "" {
		_, _ = fmt.Fprint(os.Stdout, stdout)
	}
	if stderr != "" {
		_, _ = fmt.Fprint(os.Stderr, stderr+"\n")
	}
}

func sshExec(o *optT, script string) (bool, lib.RunOut) {
	tmps := fmt.Sprintf(".__rr_src_%s", (*o).id)
	// ssh hostname 'cat - > src'
	log.Printf("CONNECTION: copying script…")
	sshenv := []string{"LC_ALL=C"}
	var ssha lib.RunArg
	var sshb lib.RunArg
	var sshc lib.RunArg
	switch len((*o).sshconfig) {
	case 0:
		switch (*o).call {
		default:
			args := []string{"-T", (*o).hostname, fmt.Sprintf("cat - > %s", tmps)}
			ssha = lib.RunArg{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte(script)}
		case cTeleport:
			args := []string{"ssh", (*o).hostname, fmt.Sprintf("cat - > %s", tmps)}
			ssha = lib.RunArg{Exe: "tsh", Args: args, Env: sshenv, Stdin: []byte(script)}
		}
	default:
		args := []string{"-F", (*o).sshconfig, "-T", (*o).hostname, fmt.Sprintf("cat - > %s", tmps)}
		ssha = lib.RunArg{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte(script)}
	}
	if ret, out := ssha.Run(); !ret {
		return ret, out
	}
	var ret bool
	var out lib.RunOut
	soFn := soOutput((*o).hostname, (*o).mode)
	switch len((*o).sshconfig) {
	case 0:
		switch (*o).sudo {
		default:
			switch (*o).call {
			default:
				args := []string{"-T", (*o).hostname, (*o).interp, tmps}
				sshb = lib.RunArg{Exe: "ssh", Args: args, Env: sshenv, Stdout: soFn}
			case cTeleport:
				args := []string{"ssh", (*o).hostname, (*o).interp, tmps}
				sshb = lib.RunArg{Exe: "tsh", Args: args, Env: sshenv, Stdout: soFn}
			}
		case cSudo:
			switch (*o).call {
			default:
				args := []string{
					"-T",
					(*o).hostname,
					"sudo",
					"-k",
					"--prompt=\"\"",
					"-S",
					"-s",
					"--",
					(*o).interp, tmps,
				}
				sshb = lib.RunArg{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte((*o).password), Stdout: soFn}
			case cTeleport:
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
				sshb = lib.RunArg{Exe: "tsh", Args: args, Env: sshenv, Stdin: []byte((*o).password), Stdout: soFn}
			}
		}
	default:
		switch (*o).sudo {
		default:
			args := []string{"-F", (*o).sshconfig, "-T", (*o).hostname, (*o).interp, tmps}
			sshb = lib.RunArg{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte((*o).password), Stdout: soFn}
		case cSudo:
			args := []string{
				"-F",
				(*o).sshconfig,
				"-T",
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
			sshb = lib.RunArg{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte((*o).password), Stdout: soFn}
		}
	}
	sshCleanUpFn := func(x bool) func(string) {
		switch len((*o).sshconfig) {
		case 0:
			switch (*o).call {
			default:
				args := []string{"-T", (*o).hostname, fmt.Sprintf("rm -f %s", tmps)}
				sshc = lib.RunArg{Exe: "ssh", Args: args, Env: sshenv}
			case cTeleport:
				args := []string{"ssh", (*o).hostname, fmt.Sprintf("rm -f %s", tmps)}
				sshc = lib.RunArg{Exe: "tsh", Args: args, Env: sshenv}
			}
		default:
			args := []string{"-F", (*o).sshconfig, "-T", (*o).hostname, fmt.Sprintf("rm -f %s", tmps)}
			sshc = lib.RunArg{Exe: "ssh", Args: args, Env: sshenv}
		}
		return func(a string) {
			if x {
				_, _ = sshc.Run()
			}
			if a != "" {
				_, _ = fmt.Fprintf(os.Stderr, "%s.", a)
			}
		}
	}
	cleanUpFn = sshCleanUpFn(true)
	// ssh hostname 'sh src'
	log.Printf("CONNECTION: running script…")
	ret, out = sshb.Run()
	// ssh hostname 'rm -f src'
	log.Printf("CONNECTION: cleaning up…")
	cleanUpFn("")
	return ret, out
}

func sudoCopy(o *optT, dir string) (bool, lib.RunOut) {
	// Three stage connection for ssh+sudo+passwd untar
	// 1. ssh hostname 'cat - > untar.sh'
	// 2. sh -c 'tar -czf - | ssh hostname 'tar -xf -'
	// 3. ssh hostname 'sudo untar.sh'
	// Why three connections? sudo STDIN is for the password
	tmpd := fmt.Sprintf(".__rr.dir.%s", (*o).id)
	tmpf := fmt.Sprintf(".__rr.tar.%s", (*o).id)
	// untar stage #3 script
	tarcmd := `
		set -efu
		export LC_ALL=C
		tar -C %s %s -cf - . | tar -C / %s -xf -
		rm -rf %s
		rm -f %s
	`
	tarexec := fmt.Sprintf(tarcmd, tmpd, cTARC, cTARX, tmpd, tmpf)
	sshenv := []string{"LC_ALL=C"}
	var untar1 lib.RunArg
	switch len((*o).sshconfig) {
	case 0:
		switch (*o).call {
		default:
			untar1 = lib.RunArg{
				Exe: "ssh",
				Args: []string{
					"-T",
					(*o).hostname,
					fmt.Sprintf("cat - > %s", tmpf),
				},
				Env:   sshenv,
				Stdin: []byte(tarexec),
			}
		case cTeleport:
			untar1 = lib.RunArg{Exe: "tsh", Args: []string{
				"ssh",
				(*o).hostname,
				fmt.Sprintf("cat - > %s", tmpf)},
				Env:   sshenv,
				Stdin: []byte(tarexec),
			}
		}
	default:
		args := []string{"-F", (*o).sshconfig, "-T", (*o).hostname, fmt.Sprintf("cat - > %s", tmpf)}
		untar1 = lib.RunArg{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte(tarexec)}
	}
	if ret, out := untar1.Run(); !ret {
		return ret, out
	}
	// untar stage #3 script
	untarDefault := `
		set -efu
		RRHOST="%s"
		RRSRC="%s"
		RRDEST="%s"
		RRSCRIPT="%s"
		tar -C "$RRSRC" %s -czf - . | ssh -T "$RRHOST" tar --one-top-level="$RRDEST" -xzf - %s
	`
	teleportDefault := `
		set -efu
		RRHOST="%s"
		RRSRC="%s"
		RRDEST="%s"
		RRSCRIPT="%s"
		tar -C "$RRSRC" %s -czf - . | tsh ssh "$RRHOST" tar --one-top-level="$RRDEST" -xzf - %s
	`
	untarConfig := `
		set -efu
		RRHOST="%s"
		RRSRC="%s"
		RRDEST="%s"
		RRCONFIG="%s"
		RRSCRIPT="%s"
		tar -C "$RRSRC" %s -czf - . | ssh -F "$RRCONFIG" -T "$RRHOST" tar --one-top-level="$RRDEST" -xzf - %s
	`
	tarenv := []string{"LC_ALL=C"}
	var untar2 lib.RunArg
	switch len((*o).sshconfig) {
	case 0:
		switch (*o).call {
		default:
			untar2 = lib.RunArg{
				Exe: (*o).interp,
				Args: []string{
					"-c",
					fmt.Sprintf(untarDefault, (*o).hostname, dir, tmpd, tmpf, cTARC, cTARX),
				},
				Env: tarenv,
			}
		case cTeleport:
			untar2 = lib.RunArg{Exe: (*o).interp, Args: []string{
				"-c",
				fmt.Sprintf(teleportDefault, (*o).hostname, dir, tmpd, tmpf, cTARC, cTARX)},
				Env: tarenv,
			}
		}
	default:
		untar2 = lib.RunArg{Exe: (*o).interp, Args: []string{
			"-c",
			fmt.Sprintf(untarConfig, (*o).hostname, dir, tmpd, (*o).sshconfig, tmpf, cTARC, cTARX)},
			Env: tarenv,
		}
	}
	if ret, out := untar2.Run(); !ret {
		return ret, out
	}
	var untar3 lib.RunArg
	switch len((*o).sshconfig) {
	case 0:
		switch (*o).call {
		default:
			args := []string{
				"-T",
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
			untar3 = lib.RunArg{
				Exe:   "ssh",
				Args:  args,
				Env:   sshenv,
				Stdin: []byte((*o).password),
			}
		case cTeleport:
			args := []string{"ssh", (*o).hostname, "sudo", "-k", "--prompt=\"\"", "-S", "-s", "--", (*o).interp, tmpf}
			untar3 = lib.RunArg{Exe: "tsh", Args: args, Env: sshenv, Stdin: []byte((*o).password)}
		}
	default:
		args := []string{"-F",
			(*o).sshconfig,
			"-T",
			(*o).hostname,
			"sudo",
			"-k",
			"--prompt=\"\"",
			"-S",
			"-s",
			"--",
			(*o).interp, tmpf,
		}
		untar3 = lib.RunArg{Exe: "ssh", Args: args, Env: sshenv, Stdin: []byte((*o).password)}
	}
	return untar3.Run()
}

func sudoCopyNopasswd(o *optT, dir string) (bool, lib.RunOut) {
	tarDefault := `
		set -efu
		tar -C %s %s -czf - . | ssh -T %s sudo -k -- tar -C / %s -xzf -
	`
	tarTeleport := `
		set -efu
		tar -C %s %s -czf - . | tsh ssh %s sudo -k -- tar -C / %s -xzf -
	`
	tarConfig := `
		set -efu
		tar -C %s %s -czf - . | ssh -F %s -T %s sudo -k -- tar -C / %s -xzf -
	`
	tarenv := []string{"LC_ALL=C"}
	var tar lib.RunArg
	switch len((*o).sshconfig) {
	case 0:
		switch (*o).call {
		default:
			tar = lib.RunArg{
				Exe:  (*o).interp,
				Args: []string{"-c", fmt.Sprintf(tarDefault, dir, cTARC, (*o).hostname, cTARX)},
				Env:  tarenv,
			}
		case cTeleport:
			tar = lib.RunArg{Exe: (*o).interp, Args: []string{
				"-c",
				fmt.Sprintf(tarTeleport, dir, cTARC, (*o).hostname, cTARX)},
				Env: tarenv,
			}
		}
	default:
		tar = lib.RunArg{Exe: (*o).interp, Args: []string{"-c",
			fmt.Sprintf(tarConfig, dir, cTARC, (*o).sshconfig, (*o).hostname, cTARX)},
			Env: tarenv,
		}
	}
	return tar.Run()
}

func quickCopy(o *optT, dir string) (bool, lib.RunOut) {
	tarDefault := `
		set -efu
		tar -C %s %s -czf - . | ssh -T %s tar -C / %s --delay-directory-restore -xzf -
	`
	tarTeleport := `
		set -efu
		tar -C %s %s -czf - . | tsh ssh %s tar -C / %s --delay-directory-restore -xzf -
	`
	tarConfig := `
		set -efu
		tar -C %s %s -czf - . | ssh -F %s -T %s tar -C / %s --delay-directory-restore -xzf -
	`
	tarenv := []string{"LC_ALL=C"}
	var tar lib.RunArg
	switch len((*o).sshconfig) {
	case 0:
		switch (*o).call {
		default:
			tar = lib.RunArg{
				Exe:  (*o).interp,
				Args: []string{"-c", fmt.Sprintf(tarDefault, dir, cTARC, (*o).hostname, cTARX)},
				Env:  tarenv,
			}
		case cTeleport:
			tar = lib.RunArg{Exe: (*o).interp, Args: []string{
				"-c",
				fmt.Sprintf(tarTeleport, dir, cTARC, (*o).hostname, cTARX)},
				Env: tarenv,
			}
		}
	default:
		tar = lib.RunArg{Exe: (*o).interp, Args: []string{"-c",
			fmt.Sprintf(tarConfig, dir, cTARC, (*o).sshconfig, (*o).hostname, cTARX)},
			Env: tarenv,
		}
	}
	return tar.Run()
}

func generateHashID() string {
	h := new(maphash.Hash)
	uid := fmt.Sprintf("%016X", h.Sum64())
	return string([]rune(uid)[:8])
}

func main() {
	runtime.MemProfileRate = 0

	var opt optT

	var failed bool = false
	var result string = "ok"

	if call := os.Args[0]; len(call) < 3 || call[len(call)-2:] == "rr" {
		log.SetOutput(io.Discard)
	} else {
		switch mode := call[len(call)-3:]; mode {
		case "rrp":
			opt.mode = cPlain
			log.SetOutput(io.Discard)
		case "rrv":
			opt.mode = cTerm
			log.SetOutput(new(logWriter))
		case "rrd":
			opt.call = cDump
			log.SetOutput(io.Discard)
		case "rrl":
			opt.call = cLog
			log.SetOutput(io.Discard)
		case "rrs":
			opt.sudopwd = cSudoPasswd
			opt.sudo = cSudo
		case "rrt":
			opt.call = cTeleport
		case "rro":
			opt.call = cTeleport
			opt.sudo = cSudo
		case "rru":
			opt.sudo = cSudo
			opt.sudopwd = cNoSudoPasswd
		default:
			valid := cPmodes
			_, _ = fmt.Fprintf(os.Stderr, "ERROR: Unsupported executable name. Valid modes:\n%s\n", lib.PipeStr("", valid))
			os.Exit(2)
		}
	}

	// rrl mode
	if cLog == opt.call {
		rrlMain()
		os.Exit(0)
	}

	log.SetFlags(0)
	if opt.mode != cPlain {
		if isatty.IsTerminal(os.Stdout.Fd()) {
			opt.mode = cTerm
			log.SetOutput(new(logWriter))
			log.Printf("taarr %s “%s”", cVERSION, cCODE)
		} else {
			serrLog = slog.New(slog.NewJSONHandler(os.Stderr, nil))
		}
	}
	// Handle top-level README
	if len(os.Args) < 2 {
		isReadme := func() (bool, string) {
			match, _ := lib.FileGlob("README*")
			for _, m := range match {
				if lib.IsFile(m) {
					return true, m
				}
			}
			return false, ""
		}
		printReadme := func(s string) {
			switch opt.mode {
			case cTerm:
				for _, each := range lib.FileLines(s) {
					fmt.Printf(" \x1b[38;2;85;85;85m⋮\x1b[0m %s\n", each)
				}
				fmt.Printf("\n")
			case cPlain:
				fmt.Print(lib.FileRead(s))
			case cJson:
				serrLog.Error("README output disabled in this mode.")
				os.Exit(2)
			}
		}
		if found1, readme1 := isReadme(); found1 && readme1 != "" {
			log.Printf("Showing %s…", readme1)
			printReadme(readme1)
			os.Exit(0)
		}
		switch opt.mode {
		case cJson:
			serrLog.Error(eUNSPECIFIED)
			os.Exit(2)
		case cTerm, cPlain:
			_, _ = fmt.Fprintln(os.Stderr, eUNSPECIFIED)
			os.Exit(2)
		}
	}

	// Handle namespace/script readmes
	{
		isReadme := func(s string) (bool, string) {
			s = strings.TrimSuffix(s, "/")
			s = strings.TrimSuffix(s, ":")
			match, _ := lib.FileGlob(fmt.Sprintf("%s/%s*", s, cDOC))
			for _, m := range match {
				if lib.IsFile(m) {
					return true, m
				}
			}
			return false, ""
		}
		printReadme := func(s string) {
			switch opt.mode {
			case cTerm:
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
				fmt.Printf(" \x1b[37;1m%s\x1b[0m │\n", pps)
				fmt.Printf("%s┘\n", line)
				for _, each := range lib.FileLines(s) {
					fmt.Printf(" \x1b[38;2;85;85;85m⋮\x1b[0m %s\n", each)
				}
				fmt.Printf("\n")
			case cPlain:
				fmt.Print(lib.FileRead(s))
			case cJson:
				serrLog.Error("README output disabled in this mode.")
				os.Exit(2)
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

	var offset int
	var hostname string
	var id string = generateHashID()
	opt.id = id // used for the random suffix in the temp filename
	if strings.Contains(os.Args[1], "/") || strings.Contains(os.Args[1], ":") {
		offset = 1
		hostname = "local"
		opt.hostname = hostname
	} else {
		offset = 2
		hostname = os.Args[1]
		opt.hostname = hostname
	}
	if len(os.Args) < offset+1 {
		switch opt.mode {
		case cTerm, cPlain:
			_, _ = fmt.Fprintln(os.Stderr, eUNSPECIFIED)
			os.Exit(2)
		case cJson:
			serrLog.Error(eUNSPECIFIED)
			os.Exit(2)
		}
	}
	jsonFile, _ := os.OpenFile(cLOG, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	defer jsonFile.Close()
	jsonLog := slog.New(slog.NewJSONHandler(jsonFile, &slog.HandlerOptions{Level: slog.LevelDebug}))
	var scr scriptT = setupScript(opt, offset)

	// rrd mode
	if cDump == opt.call {
		fmt.Print(scr.lib)
		fmt.Print(scr.code)
		os.Exit(0)
	}

	// $nsScript is the actual script to execute
	// $code is the sanitized script without rr__ variables
	var namespace string = scr.namespace
	var script string = scr.script
	var interp = scr.interp
	opt.interp = scr.interp
	var code = scr.code
	var opLog string = scr.log

	failedLogPrint := func(op string, c string, o optT, t lib.RunOut) {
		he, be, fe := conOutput(t.Stderr, "prelude", cSTDERR)
		hd, bd, fd := conOutput(t.Error, "prelude", cSTDDBG)
		b64Out := b64(t.Stdout, t.Stderr, c)
		jsonLog.Error(op, "app", "rr", "id", o.id, "code", b64Out.code, "stdout", b64Out.stdout, "stderr", b64Out.stderr, "error", t.Error)
		switch o.mode {
		case cPlain:
			stdWriter(t.Stdout, t.Stderr)
		case cJson:
			serrLog.Error(op, "stdout", t.Stdout, "stderr", t.Stderr, "error", t.Error)
		case cTerm:
			log.Printf("Failure running script!\n%s%s%s%s%s%s", he, be, fe, hd, bd, fd)
		}
	}

	okLogPrint := func(op string, c string, o optT, t lib.RunOut) {
		he, be, fe := conOutput(t.Stderr, "prelude", cSTDERR)
		hd, bd, fd := conOutput(t.Error, "prelude", cSTDDBG)
		b64Out := b64(t.Stdout, t.Stderr, c)
		jsonLog.Debug(opLog, "app", "rr", "id", o.id, "code", b64Out.code, "stdout", b64Out.stdout, "stderr", b64Out.stderr, "error", t.Error)
		jsonLog.Info(opLog, "app", "rr", "id", o.id, "result", result)
		switch o.mode {
		case cPlain:
			stdWriter(t.Stdout, t.Stderr)
		case cTerm:
			if t.Stderr != "" || t.Error != "" {
				log.Printf("Done. Output:\n%s%s%s%s%s%s", he, be, fe, hd, bd, fd)
			}
		case cJson:
			if t.Stdout != "" || t.Stderr != "" || t.Error != "" {
				serrLog.Info(opLog, "stdout", t.Stdout, "stderr", t.Stderr, "error", t.Error)
			}
		}
	}

	// Start execution routine
	jsonLog.Info(opLog, "app", "rr", "id", id, "namespace", namespace, "script", script, "target", hostname)
	if lib.IsFile(namespace + "/" + script + "/" + cPRE) {
		preStart := time.Now()
		log.Printf("Found prelude script for %s:%s. Running locally…", namespace, script)
		jsonLog.Debug("prelude", "app", "rr", "id", id, "script", script)
		soFn := soOutput("prelude", opt.mode)
		rargs := lib.RunArg{Exe: interp, Stdin: []byte(scr.prelude), Stdout: soFn}
		ret, out := rargs.Run()
		if !ret {
			failed = true
			failedLogPrint(opLog, code, opt, out)
		} else {
			okLogPrint(opLog, code, opt, out)
		}
		if tm := since(preStart); !failed {
			jsonLog.Debug(result, "app", "rr", "id", id, "start", start.Format(cTIME), "task", opLog, "target", "prelude", "namespace", namespace, "script", script, "duration", tm)
			if opt.mode == cTerm {
				log.Printf("Prelude run time: %s. Ok.", tm)
			}
		} else {
			jsonLog.Debug("failed", "app", "rr", "id", id, "start", start.Format(cTIME), "task", opLog, "target", "prelude", "namespace", namespace, "script", script, "duration", tm)
			switch opt.mode {
			case cPlain:
				// Nothing to do
			case cTerm:
				log.Printf("Prelude run time: %s. Something went wrong.", tm)
			case cJson:
				serrLog.Debug("failed", "duration", tm)
			}
			_ = jsonFile.Close()
			os.Exit(1)
		}
	}
	mainStart := time.Now()
	if opt.mode == cTerm {
		log.Printf("Running %s:%s via %s…", namespace, script, hostname)
	}
	if hostname == "local" || hostname == "localhost" {
		if opt.sudo == cSudo {
			msg := "Invoked sudo+ssh mode via local, ignored mode, just `sudo rr`."
			switch opt.mode {
			case cPlain:
				stdWriter("", msg)
			case cTerm:
				log.Print(msg)
			case cJson:
				serrLog.Error(msg)
			}
			os.Exit(2)
		}
		tar := `
			set -efu
			tar -C %s %s -cf - . | tar -C / %s --delay-directory-restore -xf -
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
			if lib.IsDir(d) {
				log.Printf("Copying %s…", d)
				jsonLog.Debug("copying", "app", "rr", "id", id, "directory", d)
				tarenv := []string{"LC_ALL=C"}
				rargs := lib.RunArg{
					Exe:  interp,
					Args: []string{"-c", fmt.Sprintf(tar, d, cTARC, cTARX)},
					Env:  tarenv,
				}
				// Error ignored because tar may fail
				_, out := rargs.Run()
				b64Out := b64(out.Stdout, out.Stderr, "")
				jsonLog.Debug("copy", "app", "rr", "id", id, "stdout", b64Out.stdout, "stderr", b64Out.stderr, "error", out.Error)
				jsonLog.Info("copy", "app", "rr", "id", id, "result", "finished")
				if opt.mode == cTerm {
					log.Printf("Finished copying files")
				}
			}
		}
		if opLog == "UNDEFINED" {
			log.Printf("Running %s…", script)
			jsonLog.Debug("running", "app", "rr", "id", id, "script", script)
		} else {
			msgop := strings.TrimSuffix(opLog, "\n")
			log.Printf("%s…", msgop)
			jsonLog.Debug(msgop, "app", "rr", "id", id, "script", script)
		}
		soFn := soOutput(hostname, opt.mode)
		rargs := lib.RunArg{Exe: interp, Stdin: []byte(scr.nsscript), Stdout: soFn}
		ret, out := rargs.Run()
		if !ret {
			failed = true
			failedLogPrint(opLog, code, opt, out)
		} else {
			scanner := bufio.NewScanner(strings.NewReader(out.Stderr))
			scanner.Split(bufio.ScanWords)
			for scanner.Scan() {
				if strings.Contains(scanner.Text(), cREPAIRED) {
					result = "repaired"
				}
			}
			okLogPrint(opLog, code, opt, out)
		}
	} else if _, err := strconv.ParseInt(hostname, 10, 64); err == nil {
		destination := fmt.Sprintf("/proc/%s/root", hostname)
		for _, d := range []string{
			".files/",
			namespace + "/.files/",
			namespace + "/" + script + "/.files/",
		} {
			if lib.IsDir(d) {
				log.Printf("Copying %s…", d)
				jsonLog.Debug("copying", "app", "rr", "id", id, "directory", d)
				tarenv := []string{"LC_ALL=C"}
				tar := `
					set -efu
					tar -C %s %s -cf - . | tar -C %s %s -xf -
				`
				rsargs := lib.RunArg{
					Exe:  interp,
					Args: []string{"-c", fmt.Sprintf(tar, d, cTARC, destination, cTARX)},
					Env:  tarenv,
				}
				ret, out := rsargs.Run()
				b64Out := b64(out.Stdout, out.Stderr, "")
				if step := "copy"; !ret {
					jsonLog.Error(step, "app", "rr", "id", id, "stdout", b64Out.stdout, "stderr", b64Out.stderr, "error", out.Error)
					switch opt.mode {
					case cPlain:
						stdWriter(out.Stdout, out.Stderr)
					case cJson:
						serrLog.Error(step, "stdout", out.Stdout, "stderr", out.Stderr, "error", out.Error)
					case cTerm:
						ho, bo, fo := conOutput(out.Stdout, hostname, cSTDOUT)
						he, be, fe := conOutput(out.Stderr, hostname, cSTDERR)
						hd, bd, fd := conOutput(out.Error, hostname, cSTDDBG)
						log.Printf("Error encountered.\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
						log.Printf("Failure copying files!")
					}
					os.Exit(1)
				} else {
					jsonLog.Debug(step, "app", "rr", "id", id, "stdout", b64Out.stdout, "stderr", b64Out.stderr, "error", out.Error)
					jsonLog.Info(step, "app", "rr", "id", id, "result", "copied")
					if opt.mode == cTerm {
						log.Printf("Finished copying files")
					}
				}
			}
		}
		log.Printf("Running %s…", script)
		jsonLog.Debug("running", "app", "rr", "id", id, "script", script)
		soFn := soOutput(hostname, opt.mode)
		nsargs := lib.RunArg{Exe: "nsenter", Args: []string{"-a", "-r", "-t", hostname, interp, "-c", scr.nsscript}, Stdout: soFn}
		ret, out := nsargs.Run()
		if !ret {
			failed = true
			failedLogPrint(opLog, code, opt, out)
		} else {
			scanner := bufio.NewScanner(strings.NewReader(out.Stderr))
			scanner.Split(bufio.ScanWords)
			for scanner.Scan() {
				if strings.Contains(scanner.Text(), cREPAIRED) {
					result = "repaired"
				}
			}
			okLogPrint(opLog, code, opt, out)
		}
	} else {
		if opt.call != cTeleport {
			switch {
			case lib.IsFile(cHOSTS1):
				opt.sshconfig = cHOSTS1
			case lib.IsFile(cHOSTS2):
				opt.sshconfig = cHOSTS2
			default:
				opt.sshconfig = cHOSTS0
			}
		}
		var realhost string
		if rh := strings.Split(hostname, "@"); len(rh) == 1 {
			realhost = hostname
		} else {
			realhost = rh[1]
		}
		for _, d := range []string{
			".files",
			".files-" + realhost,
			namespace + "/.files",
			namespace + "/.files-" + realhost,
			namespace + "/" + script + "/.files",
			namespace + "/" + script + "/.files-" + realhost,
		} {
			if lib.IsDir(d) {
				jsonLog.Debug("copying", "app", "rr", "id", id, "directory", d)
				log.Printf("CONNECTION: copying %s to %s…", d, realhost)
				var ret bool
				var out lib.RunOut
				switch opt.sudo {
				default:
					// Error ignored because tar may fail
					_, out = quickCopy(&opt, d)
					ret = true
				case cSudo:
					if opt.sudopwd == cSudoPasswd {
						ret, out = sudoCopy(&opt, d)
					} else {
						ret, out = sudoCopyNopasswd(&opt, d)
					}
				}
				b64Out := b64(out.Stdout, out.Stderr, "")
				if step := "copy"; !ret && opt.sudo == cSudo {
					jsonLog.Error("step", "app", "rr", "id", id, "stdout", b64Out.stdout, "stderr", b64Out.stderr, "error", out.Error)
					switch opt.mode {
					case cPlain:
						stdWriter(out.Stdout, out.Stderr)
					case cTerm:
						ho, bo, fo := conOutput(out.Stdout, hostname, cSTDOUT)
						he, be, fe := conOutput(out.Stderr, hostname, cSTDERR)
						hd, bd, fd := conOutput(out.Error, hostname, cSTDDBG)
						log.Printf("Error encountered.\n%s%s%s%s%s%s%s%s%s", ho, bo, fo, he, be, fe, hd, bd, fd)
						log.Printf("Failure copying files!")
					case cJson:
						serrLog.Error(step, "stdout", out.Stdout, "stderr", out.Stderr, "error", out.Error)
					}
					os.Exit(1)
				} else {
					jsonLog.Debug(step, "app", "rr", "id", id, "stdout", b64Out.stdout, "stderr", b64Out.stderr, "error", out.Error)
					jsonLog.Info(step, "app", "rr", "id", id, "result", "copied")
					if opt.mode == cTerm {
						log.Printf("Finished copying files")
					}
				}
			}
		}
		if opt.mode == cTerm {
			log.Printf("Running %s…", script)
		}
		jsonLog.Debug("running", "app", "rr", "id", id, "script", script)
		var ret bool
		var out lib.RunOut
		if opt.sudo == cSudo {
			if opt.sudopwd == cSudoPasswd {
				str, err := getPassword("sudo password: ")
				if err != nil {
					switch opt.mode {
					case cTerm, cPlain:
						_, _ = fmt.Fprintf(os.Stderr, "Unable to initialize STDIN or this is not a terminal.\n")
						os.Exit(2)
					case cJson:
						serrLog.Error("Unable to initialize STDIN or this is not a terminal.", "namespace", namespace, "script", script)
						os.Exit(2)
					}
				}
				opt.password = fmt.Sprintf("%s\n", str)
			}
		}
		ret, out = sshExec(&opt, scr.nsscript)
		if !ret {
			failed = true
			failedLogPrint(opLog, code, opt, out)
		} else {
			scanner := bufio.NewScanner(strings.NewReader(out.Stderr))
			scanner.Split(bufio.ScanWords)
			for scanner.Scan() {
				if strings.Contains(scanner.Text(), cREPAIRED) {
					result = "repaired"
				}
			}
			okLogPrint(opLog, code, opt, out)
		}
	}
	if tm := since(mainStart); !failed {
		jsonLog.Debug(result, "app", "rr", "id", id, "start", start.Format(cTIME), "task", opLog, "target", hostname, "namespace", namespace, "script", script, "duration", tm)
		if opt.mode == cTerm {
			log.Printf("Run time: %s. Ok.", tm)
		}
	} else {
		jsonLog.Debug("failed", "app", "rr", "id", id, "start", start.Format(cTIME), "task", opLog, "target", hostname, "namespace", namespace, "script", script, "duration", tm)
		switch opt.mode {
		case cPlain:
			// Nothing to do
		case cTerm:
			log.Printf("Run time: %s. Something went wrong.", tm)
		case cJson:
			serrLog.Debug("failed", "duration", tm)
		}
		_ = jsonFile.Close()
		os.Exit(1)
	}
	if lib.IsFile(namespace + "/" + script + "/" + cPOST) {
		postStart := time.Now()
		log.Printf("Found epilogue script for %s:%s. Running locally…", namespace, script)
		jsonLog.Debug("epilogue", "app", "rr", "id", id, "script", script)
		soFn := soOutput("epilogue", opt.mode)
		rargs := lib.RunArg{Exe: interp, Stdin: []byte(scr.epilogue), Stdout: soFn}
		ret, out := rargs.Run()
		if !ret {
			failed = true
			failedLogPrint(opLog, code, opt, out)
		} else {
			okLogPrint(opLog, code, opt, out)
		}
		if tm := since(postStart); !failed {
			jsonLog.Debug(result, "app", "rr", "id", id, "start", start.Format(cTIME), "task", opLog, "target", "epilogue", "namespace", namespace, "script", script, "duration", tm)
			if opt.mode == cTerm {
				log.Printf("Epilogue run time: %s. Ok.", tm)
			}
		} else {
			jsonLog.Debug("failed", "app", "rr", "id", id, "start", start.Format(cTIME), "task", opLog, "target", "epilogue", "namespace", namespace, "script", script, "duration", tm)
			switch opt.mode {
			case cPlain:
				// Nothing to do
			case cTerm:
				log.Printf("Epilogue run time: %s. Something went wrong.", tm)
			case cJson:
				serrLog.Debug("failed", "duration", tm)
			}
			_ = jsonFile.Close()
			os.Exit(1)
		}
	}
	if tm := since(start); opt.mode == cTerm && (0 != len(scr.prelude) || 0 != len(scr.epilogue)) {
		log.Printf("Total run time: %s. All OK.", tm)
	}
	_ = jsonFile.Close()
	os.Exit(0)
}
