package proto

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Pasta dos dados do jogo: uma raiz "AsciiMMO" (por padrão em Documentos),
// com "save" para os personagens do servidor e "cliente" para as opções e o
// mapa explorado. O jogador pode apontar outra raiz no painel de opções; a
// escolha fica num arquivo pequeno num lugar fixo (a pasta de configuração
// do sistema), que o servidor e o cliente leem ao abrir.

const GameName = "AsciiMMO"

// pointerFile guarda a raiz escolhida pelo jogador.
func pointerFile() string {
	d, err := os.UserConfigDir()
	if err != nil {
		d = "."
	}
	return filepath.Join(d, GameName, "pasta.txt")
}

// DefaultRoot é Documentos/AsciiMMO (ou a pasta do usuário, se não houver Documentos).
func DefaultRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return GameName
	}
	if docs := filepath.Join(home, "Documents"); dirExists(docs) {
		return filepath.Join(docs, GameName)
	}
	return filepath.Join(home, GameName)
}

// DataRoot é a raiz em uso: a escolhida pelo jogador ou a padrão.
func DataRoot() string {
	if b, err := os.ReadFile(pointerFile()); err == nil {
		if s := strings.TrimSpace(string(b)); s != "" {
			return s
		}
	}
	return DefaultRoot()
}

// SetDataRoot grava a raiz escolhida e leva para lá o que já existia.
func SetDataRoot(root string) error {
	old := DataRoot()
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(pointerFile()), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(pointerFile(), []byte(root), 0o644); err != nil {
		return err
	}
	if !samePath(old, root) {
		CopyMissing(old, root)
	}
	return nil
}

// SaveDir é onde o servidor guarda os personagens.
func SaveDir() string { return filepath.Join(DataRoot(), "save") }

// ClientDir é onde o cliente guarda as opções e o mapa explorado.
func ClientDir() string { return filepath.Join(DataRoot(), "cliente") }

// CopyMissing copia de src para dst, em profundidade, o que ainda não existe
// em dst. Não apaga nada da origem.
func CopyMissing(src, dst string) {
	filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return nil
		}
		t := filepath.Join(dst, rel)
		if info.IsDir() {
			os.MkdirAll(t, 0o755)
			return nil
		}
		if _, err := os.Stat(t); err == nil {
			return nil
		}
		in, err := os.Open(p)
		if err != nil {
			return nil
		}
		defer in.Close()
		out, err := os.Create(t)
		if err != nil {
			return nil
		}
		defer out.Close()
		io.Copy(out, in)
		return nil
	})
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func samePath(a, b string) bool {
	x, _ := filepath.Abs(a)
	y, _ := filepath.Abs(b)
	return strings.EqualFold(filepath.Clean(x), filepath.Clean(y))
}
