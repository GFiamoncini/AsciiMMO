package main

import (
	"os/exec"
	"strings"
	"syscall"
)

// chooseFolder mostra a janela de escolher pasta do Windows (pelo PowerShell,
// sem console aparecendo) e devolve a pasta escolhida, ou "" se cancelou.
func chooseFolder(start string) (string, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms;` +
		`$f = New-Object System.Windows.Forms.FolderBrowserDialog;` +
		`$f.Description = 'Pasta dos dados do AsciiMMO';` +
		`$f.SelectedPath = '` + strings.ReplaceAll(start, "'", "''") + `';` +
		`if ($f.ShowDialog() -eq 'OK') { $f.SelectedPath }`
	cmd := exec.Command("powershell", "-NoProfile", "-STA", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}
