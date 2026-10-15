package main

import (
	"syscall"
	"unsafe"
)

// darkTitleBar pinta a barra de título da janela de escuro, na cor do jogo
// (o Windows deixa branca por padrão). Devolve false se a janela ainda não
// existe, para tentar de novo no próximo quadro.
func darkTitleBar(title string) bool {
	user32 := syscall.NewLazyDLL("user32.dll")
	dwm := syscall.NewLazyDLL("dwmapi.dll")
	find := user32.NewProc("FindWindowW")
	set := dwm.NewProc("DwmSetWindowAttribute")
	if find.Find() != nil || set.Find() != nil {
		return true // sistema sem essas funções: não insiste
	}
	t, _ := syscall.UTF16PtrFromString(title)
	hwnd, _, _ := find.Call(0, uintptr(unsafe.Pointer(t)))
	if hwnd == 0 {
		return false
	}
	on := int32(1)
	set.Call(hwnd, 20, uintptr(unsafe.Pointer(&on)), 4)      // DWMWA_USE_IMMERSIVE_DARK_MODE
	caption := uint32(0x0f0909)                              // COLORREF em BGR: #09090f, o fundo do jogo
	set.Call(hwnd, 35, uintptr(unsafe.Pointer(&caption)), 4) // DWMWA_CAPTION_COLOR (Windows 11)
	text := uint32(0xc8d0e6)
	set.Call(hwnd, 36, uintptr(unsafe.Pointer(&text)), 4) // DWMWA_TEXT_COLOR
	return true
}
