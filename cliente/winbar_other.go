//go:build !windows

package main

// darkTitleBar só faz algo no Windows; nos outros sistemas a barra segue o tema.
func darkTitleBar(string) bool { return true }
