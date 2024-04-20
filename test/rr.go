package rr

import "os"

const cLIB = ".lib/999-test.sh"
const cRR = "../bin/rr"

func mkTemp() string {
	f, _ := os.CreateTemp("", "rr_*")
	name := f.Name()
	defer f.Close()
	defer os.Remove(name)
	return name
}
