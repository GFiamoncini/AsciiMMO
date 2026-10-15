//go:build !windows

package main

import (
	"os/exec"
	"strings"
)

// chooseFolder usa o zenity, se houver; senão não faz nada.
func chooseFolder(start string) (string, error) {
	out, err := exec.Command("zenity", "--file-selection", "--directory", "--filename="+start+"/").Output()
	return strings.TrimSpace(string(out)), err
}
