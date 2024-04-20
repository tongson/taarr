package rr

import (
	"fmt"
	"os"
	"strings"
	"testing"

	. "github.com/tongson/gl"
)

const cLIB = ".lib/999-test.sh"
const cEXE = "../bin/rr"

func TestMain(m *testing.M) {
	code := m.Run()
	os.Remove("LOG")
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

func TestOp(T *testing.T) {
	T.Parallel()
	msg := "Somebody set up us the bomb"
	T.Run("environment", func(t *testing.T) {
		env := []string{fmt.Sprintf("OP=%s", msg)}
		rr := RunArg{Exe: cEXE, Args: []string{"op:env"}, Env: env}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
		if got := strings.Contains(FileRead("LOG"), msg); !got {
			t.Error("wants `true`")
		}
	})
	T.Run("file", func(t *testing.T) {
		StringToFile("OP", msg)
		rr := RunArg{Exe: cEXE, Args: []string{"op:file"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
		if got := strings.Contains(FileRead("LOG"), msg); !got {
			t.Error("wants `true`")
		}
		t.Cleanup(func() {
			os.Remove("OP")
		})
	})
}

func TestRepaired(T *testing.T) {
	T.Parallel()
	T.Run("repaired1", func(t *testing.T) {
		rr := RunArg{Exe: cEXE, Args: []string{"repaired:nolinebreak"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
		if got := strings.Contains(FileRead("LOG"), "\"message\":\"repaired\""); !got {
			t.Error("wants `true`")
		}
	})
	T.Run("repaired2", func(t *testing.T) {
		rr := RunArg{Exe: cEXE, Args: []string{"repaired:linebreak"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
		if got := strings.Contains(FileRead("LOG"), "\"message\":\"repaired\""); !got {
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
	// XXX: Might remove support for calls like this.
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
	T.Run("readme1", func(t *testing.T) {
		rr := RunArg{Exe: cEXE, Args: []string{"readme/check"}}
		if x, o := rr.Run(); !x {
			t.Error("wants `true`")
		} else {
			stdout := o.Stdout
			if got := strings.Split(stdout, "\n")[3]; got != "TEST" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
			if got := strings.Split(stdout, "\n")[4]; got != "README" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
		}

	})
	T.Run("readme2", func(t *testing.T) {
		rr := RunArg{Exe: cEXE, Args: []string{"readme/check/"}}
		if x, o := rr.Run(); !x {
			t.Error("wants `true`")
		} else {
			stdout := o.Stdout
			if got := strings.Split(stdout, "\n")[3]; got != "TEST" {
				t.Errorf("Unexpected STDOUT: `%s`\n", got)
			}
			if got := strings.Split(stdout, "\n")[4]; got != "README" {
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
