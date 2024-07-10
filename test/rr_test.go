package rr

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	. "github.com/tongson/gl"
)

const cLIB = "VARS"
const cEXE = "../bin/rr"

func TestMain(m *testing.M) {
	code := m.Run()
	os.Remove("LOG")
	nsenter := `
	sudo ../bin/rr $(pgrep sshd) remote:simple
	sudo ../bin/rr $(pgrep sshd) files:remote
!!!!
	`
	print("!!!!\nConsider nsenter tests:\n")
	print(nsenter)
	os.Exit(code)
}

func TestRun(T *testing.T) {
	T.Parallel()
	T.Run("gl_simple1", func(t *testing.T) {
		exe := RunArg{Exe: "true"}
		if ret, _ := exe.Run(); !ret {
			t.Error("wants `true`")
		}
	})
	T.Run("gl_simple2", func(t *testing.T) {
		exe := RunArg{Exe: "false"}
		if ret, _ := exe.Run(); ret {
			t.Error("wants `false`")
		}
	})
}

func TestEnv(t *testing.T) {
	testenv := []string{"rr__LOOKFORTHIS=FOO"}
	rr := RunArg{Exe: cEXE + "p", Env: testenv, Args: []string{"env:test"}}
	ret, out := rr.Run()
	if !ret {
		t.Error("wants `true`")
	}
	re := regexp.MustCompile("(?m)^LOOKFORTHIS.*$")
	match := re.FindAllString(out.Stdout, -1)
	var ok bool
	for _, s := range match {
		if s == "LOOKFORTHIS=FOO" {
			ok = true
		}
	}
	if !ok {
		t.Error("wants `true`")
	}
}

func TestOp(T *testing.T) {
	T.Parallel()
	msg := "Somebody set up us the bomb"
	T.Run("environment", func(t *testing.T) {
		env := []string{fmt.Sprintf("LOG=%s", msg)}
		rr := RunArg{Exe: cEXE, Args: []string{"op:env"}, Env: env}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
		if got := strings.Contains(FileRead("LOG"), msg); !got {
			t.Error("wants `true`")
		}
	})
	T.Run("argument", func(t *testing.T) {
		rr := RunArg{Exe: cEXE, Args: []string{"op:env", "__argument__"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
		if got := strings.Contains(FileRead("LOG"), "__argument__"); !got {
			t.Error("wants `true`")
		}
	})
}

func TestRepaired(T *testing.T) {
	T.Parallel()
	T.Run("repaired1", func(t *testing.T) {
		rr := RunArg{Exe: cEXE, Args: []string{"repaired:nolinebreak"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
		if got := strings.Contains(FileRead("LOG"), "\"msg\":\"repaired\""); !got {
			t.Error("wants `true`")
		}
	})
	T.Run("repaired2", func(t *testing.T) {
		rr := RunArg{Exe: cEXE, Args: []string{"repaired:linebreak"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
		if got := strings.Contains(FileRead("LOG"), "\"msg\":\"repaired\""); !got {
			t.Error("wants `true`")
		}
	})
}

func TestArgs(T *testing.T) {
	T.Parallel()
	T.Run("args1", func(t *testing.T) {
		one := mkTemp()
		two := mkTemp()
		three := mkTemp()
		StringToFile(cLIB, fmt.Sprintf("ONE=%s\nTWO=%s\nTHREE=%s\n", one, two, three))
		rr := RunArg{Exe: cEXE, Args: []string{"args:args1", "--one", "--two", "--three"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
		if got := FileRead(one); got != "--one\n" {
			t.Errorf("Unexpected contents of %s: `%s`\n", one, got)
		}
		if got := FileRead(two); got != "--two\n" {
			t.Errorf("Unexpected contents of %s: `%s`\n", two, got)
		}
		if got := FileRead(three); got != "--three\n" {
			t.Errorf("Unexpected contents of %s: `%s`\n", three, got)
		}
		t.Cleanup(func() {
			os.Remove(cLIB)
			os.Remove(one)
			os.Remove(two)
			os.Remove(three)
		})
	})
	T.Run("args2", func(t *testing.T) {
		one := mkTemp()
		StringToFile(cLIB, fmt.Sprintf("ONE=%s\n", one))
		rr := RunArg{Exe: cEXE, Args: []string{"args:args2", "one", "1"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
		if got := FileRead(one); got != "1\n" {
			t.Errorf("Unexpected contents of %s: `%s`\n", one, got)
		}
		t.Cleanup(func() {
			os.Remove(cLIB)
			os.Remove(one)
		})
	})
	T.Run("args3", func(t *testing.T) {
		vee := mkTemp()
		StringToFile(cLIB, fmt.Sprintf("TEMPFILE=%s\n", vee))
		rr := RunArg{Exe: cEXE, Args: []string{"args:args3", "-v"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
		if got := FileRead(vee); got != "v\n" {
			t.Errorf("Unexpected contents of %s: `%s`\n", vee, got)
		}
		t.Cleanup(func() {
			os.Remove(cLIB)
			os.Remove(vee)
		})
	})
	// XXX Might remove support for calls like this.
	T.Run("args4a", func(t *testing.T) {
		rr := RunArg{Exe: cEXE, Args: []string{"args:args4:1"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
	})
	T.Run("args4b", func(t *testing.T) {
		rr := RunArg{Exe: cEXE, Args: []string{"local", "args:args4:1"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
	})
	T.Run("args4c", func(t *testing.T) {
		rr := RunArg{Exe: cEXE, Args: []string{"args:args6:1", "2", "3", "4"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
	})
}

func TestReadme(T *testing.T) {
	T.Parallel()
	T.Run("readme1a", func(t *testing.T) {
		rr := RunArg{Exe: cEXE + "p", Args: []string{"readme"}}
		if x, o := rr.Run(); !x {
			t.Error("wants `true`")
		} else {
			stdout := o.Stdout
			if got := strings.Split(stdout, "\n")[0]; got != "TEST" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
			if got := strings.Split(stdout, "\n")[1]; got != "README1" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
		}

	})
	T.Run("readme1b", func(t *testing.T) {
		rr := RunArg{Exe: cEXE + "p", Args: []string{"readme/"}}
		if x, o := rr.Run(); !x {
			t.Error("wants `true`")
		} else {
			stdout := o.Stdout
			if got := strings.Split(stdout, "\n")[0]; got != "TEST" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
			if got := strings.Split(stdout, "\n")[1]; got != "README1" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
		}

	})
	T.Run("readme2a", func(t *testing.T) {
		rr := RunArg{Exe: cEXE + "p", Args: []string{"readme/check"}}
		if x, o := rr.Run(); !x {
			t.Error("wants `true`")
		} else {
			stdout := o.Stdout
			if got := strings.Split(stdout, "\n")[0]; got != "TEST" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
			if got := strings.Split(stdout, "\n")[1]; got != "README2" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
		}

	})
	T.Run("readme2b", func(t *testing.T) {
		rr := RunArg{Exe: cEXE + "p", Args: []string{"readme/check/"}}
		if x, o := rr.Run(); !x {
			t.Error("wants `true`")
		} else {
			stdout := o.Stdout
			if got := strings.Split(stdout, "\n")[0]; got != "TEST" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
			if got := strings.Split(stdout, "\n")[1]; got != "README2" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
		}

	})
}

func TestInterpreter(T *testing.T) {
	T.Parallel()
	T.Run("interpreter/python", func(t *testing.T) {
		// One directory down. CWD is `interpreter`.
		// ../../bin/rrp to force output.
		rr := RunArg{Exe: "../" + cEXE + "p", Args: []string{"shell:python"}, Dir: "interpreter"}
		if x, o := rr.Run(); !x {
			t.Error("wants `true`")
		} else {
			if got := o.Stdout; got != "__main__\n" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
		}
	})
}

func TestPrelude(T *testing.T) {
	T.Parallel()
	T.Run("prelude ok", func(t *testing.T) {
		rr := RunArg{Exe: cEXE + "p", Args: []string{"prelude:ok"}}
		if x, o := rr.Run(); !x {
			t.Error("wants `true`")
		} else {
			if got := o.Stdout; got != "prelude\nscript\n" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
		}
	})
	T.Run("prelude fail", func(t *testing.T) {
		rr := RunArg{Exe: cEXE + "p", Args: []string{"prelude:fail"}}
		if x, o := rr.Run(); x {
			t.Error("wants `false`")
		} else {
			if got := o.Stdout; got != "prelude\n" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
		}
	})
}

func TestEpilogue(T *testing.T) {
	T.Parallel()
	T.Run("epilogue ok", func(t *testing.T) {
		rr := RunArg{Exe: cEXE + "p", Args: []string{"epilogue:ok"}}
		if x, o := rr.Run(); !x {
			t.Error("wants `true`")
		} else {
			if got := o.Stdout; got != "script\nepilogue\n" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
		}
	})
	T.Run("epilogue fail", func(t *testing.T) {
		rr := RunArg{Exe: cEXE + "p", Args: []string{"epilogue:fail"}}
		if x, o := rr.Run(); x {
			t.Error("wants `false`")
		} else {
			if got := o.Stdout; got != "script\nepilogue\n" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
		}
	})
	T.Run("epilogue main fail", func(t *testing.T) {
		rr := RunArg{Exe: cEXE + "p", Args: []string{"epilogue:main_fail"}}
		if x, o := rr.Run(); x {
			t.Error("wants `false`")
		} else {
			if got := o.Stdout; got != "script\n" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
		}
	})
}

func TestPlain(T *testing.T) {
	T.Run("plain mode", func(t *testing.T) {
		rr := RunArg{Exe: "sh", Args: []string{"-c", "../bin/rrp plain:main $(../bin/rrp plain:arg)"}}
		if x, o := rr.Run(); !x {
			t.Error("wants `true`")
		} else {
			if got := o.Stdout; got != "one\n" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
		}
	})
}

func TestSsh(T *testing.T) {
	T.Parallel()
	T.Run("simple", func(t *testing.T) {
		rr := RunArg{Exe: cEXE, Args: []string{"foo@chroot", "remote:simple"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
	})
	T.Run("sudo+nopasswd", func(t *testing.T) {
		rr := RunArg{Exe: cEXE + "u", Args: []string{"bar@chroot", "remote:nopasswd"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
	})
	T.Run("simple+fail", func(t *testing.T) {
		rr := RunArg{Exe: cEXE, Args: []string{"foo@chroot", "remote:simple-fail"}}
		if ret, _ := rr.Run(); ret {
			t.Error("wants `false`")
		}
	})
	T.Run("sudo+nopasswd+fail", func(t *testing.T) {
		rr := RunArg{Exe: cEXE + "u", Args: []string{"bar@chroot", "remote:nopasswd-fail"}}
		if ret, _ := rr.Run(); ret {
			t.Error("wants `false`")
		}
	})
}

func TestLocalFiles(T *testing.T) {
	T.Parallel()
	T.Run("local", func(t *testing.T) {
		x := "/tmp/__rr_FFF"
		y := "/tmp/__rr_YYY"
		z := "/tmp/__rr_ZZZ"
		rr := RunArg{Exe: cEXE, Args: []string{"files:local"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
		if !IsDir(x) {
			t.Error("wants `true`")
		}
		if !IsDir(y) {
			t.Error("wants `true`")
		}
		if !IsFile(z) {
			t.Error("wants `true`")
		}
		t.Cleanup(func() {
			os.Remove(x)
			os.Remove(y)
			os.Remove(z)
		})
	})
}

func TestSshFiles(T *testing.T) {
	// Some overlapping files. Run serially.
	T.Run("remote", func(t *testing.T) {
		rr := RunArg{Exe: cEXE, Args: []string{"foo@chroot", "files:remote"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
	})
	T.Run("remote+sudo+nopasswd", func(t *testing.T) {
		rr := RunArg{Exe: cEXE + "u", Args: []string{"bar@chroot", "files:remote-nopasswd"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
	})
}

func TestFail(T *testing.T) {
	T.Parallel()
	T.Run("sudo_wo_stdin", func(t *testing.T) {
		rr := RunArg{Exe: cEXE + "s", Args: []string{"args:args1"}}
		ret, out := rr.Run()
		if ret {
			t.Error("wants `false`")
		}
		if out.Stderr[0:1] != "{" {
			t.Errorf("Unexpected output: %s\n", out.Stderr)
		}
	})
}
