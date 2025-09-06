package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed make/libs/.lib/**
var libs embed.FS

func extractFsys(fsys embed.FS, fsroot string, destDir string) error {
	return fs.WalkDir(fsys, fsroot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(fsroot, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(destDir, relPath)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0o700)
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, 0o644)
	})
}

func rrfMain() {
	if err := os.Mkdir(".lib", 0o700); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to create the directory `%s` in the current directory. The directory may already exist or you may not have write permissions. Please remove or rename the existing directory.\n", ".lib")
		os.Exit(cExitCantCreate)
	}
	if err := extractFsys(libs, "make/libs/.lib", ".lib"); err != nil {
		_, _ = fmt.Fprint(os.Stderr, "Failed to extract files.\n")
		os.Exit(cExitOsErr)
	}
}
