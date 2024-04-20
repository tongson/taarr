package rr

import "os"

func mkTemp() string {
	f, _ := os.CreateTemp("", "rr_*")
	name := f.Name()
	defer f.Close()
	defer os.Remove(name)
	return name
}
