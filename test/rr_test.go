package rr

import (
	"fmt"
	"os"
	"testing"

	. "github.com/tongson/gl"
)

const cLIB = ".lib/999-test.sh"
const cEXE = "../bin/rr"

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

func TestArgs(T *testing.T) {
	T.Parallel()
	T.Run("args1", func(t *testing.T) {
		one := mkTemp()
		two := mkTemp()
		three := mkTemp()
		vars := fmt.Sprintf("ONE=%s\nTWO=%s\nTHREE=%s\n", one, two, three)
		StringToFile(cLIB, vars)
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
		vars := fmt.Sprintf("ONE=%s\n", one)
		StringToFile(cLIB, vars)
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
		rr := RunArg{Exe: cEXE, Args: []string{"args:args3", "-v"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
	})
	T.Run("args4", func(t *testing.T) {
		rr := RunArg{Exe: cEXE, Args: []string{"args:args4:1"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
	})
	T.Run("args5", func(t *testing.T) {
		rr := RunArg{Exe: cEXE, Args: []string{"local", "args:args4:1"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
	})
	T.Run("args6", func(t *testing.T) {
		rr := RunArg{Exe: cEXE, Args: []string{"args:args6:1", "2", "3", "4"}}
		if ret, _ := rr.Run(); !ret {
			t.Error("wants `true`")
		}
	})

}
