package main

import (
	"math"
	"testing"
	"time"
)

// TestSintese confere que todos os sons gerados existem, têm duração, não
// têm NaN nem estouro, e que a geração cabe na abertura do jogo.
func TestSintese(t *testing.T) {
	t0 := time.Now()
	sfx, loops := genSounds(), genLoops()
	if d := time.Since(t0); d > 4*time.Second {
		t.Errorf("gerar os sons levou %v", d)
	}
	check := func(name string, w wave) {
		if len(w) < sampleRate/100 {
			t.Errorf("%s curto demais: %d amostras", name, len(w))
		}
		peak, sum := 0.0, 0.0
		for _, v := range w {
			f := float64(v)
			if math.IsNaN(f) || math.IsInf(f, 0) {
				t.Fatalf("%s tem NaN/Inf", name)
			}
			peak = math.Max(peak, math.Abs(f))
			sum += f * f
		}
		if peak > 1.001 || peak < 0.05 {
			t.Errorf("%s com pico %.3f", name, peak)
		}
		if math.Sqrt(sum/float64(len(w))) < 0.003 {
			t.Errorf("%s quase mudo", name)
		}
	}
	for n, ws := range sfx {
		for _, w := range ws {
			check(n, w)
		}
	}
	for n, w := range loops {
		check(n, w)
	}
	// Todo nome usado pelos eventos tem de existir.
	used := []string{"click", "hurt", "death", "levelup", "coin", "pickup", "bonfire", "msg", "tree_fall", "wood_break",
		"stone_break", "crystal_break", "hit", "hit_bone", "hit_stone", "swing", "splash", "step_grass", "step_stone",
		"step_mud", "step_snow", "bird", "owl", "frog", "drip", "wolf", "roar"}
	for _, n := range fxSound {
		used = append(used, n)
	}
	for _, n := range projSound {
		used = append(used, n)
	}
	used = append(used, mobVoice...)
	used = append(used, steedVoice...)
	for _, n := range used {
		if n != "" && len(sfx[n]) == 0 {
			t.Errorf("som %q usado mas não gerado", n)
		}
	}
	for _, n := range []string{"wind", "crickets", "rain", "sea", "cave", "lava", "fire"} {
		if loops[n] == nil {
			t.Errorf("laço %q não gerado", n)
		}
	}
	for n := range musicBase {
		if _, err := musicFS.ReadFile("sons/musica/" + n + ".ogg"); err != nil {
			t.Errorf("música %s: %v", n, err)
		}
	}
}
