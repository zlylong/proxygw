package main

import (
	"os"
	"path/filepath"
	"strings"
)

func getAppRoot() string {
	if root := os.Getenv("PROXYGW_HOME"); root != "" {
		return root
	}
	return "/root/proxygw"
}

func getPath(elem ...string) string {
	paths := append([]string{getAppRoot()}, elem...)
	return filepath.Join(paths...)
}

func getRelativePath(p string) string {
	if strings.HasPrefix(p, "../") {
		return filepath.Join(getAppRoot(), strings.TrimPrefix(p, "../"))
	}
	return p
}
