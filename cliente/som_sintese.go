package main

import "math"

// Sons gerados por código: golpes, magias, quebras, bichos e os laços de
// ambiente (vento, grilos, chuva, mar, caverna, lava, fogo). Tudo sai de
// ruído e osciladores com filtros simples, uma vez só, na abertura do jogo,
// como o ícone e os glifos: nada de arquivo para os efeitos. A música, essa
// sim, vem em ogg (som.go).

const sampleRate = 44100

// wave é um som mono em ponto flutuante, de -1 a 1.
type wave []float32

func secs(s float64) wave { return make(wave, int(s*sampleRate)) }

// rnd é um gerador pequeno e determinístico (o mesmo som a cada abertura).
type rnd struct{ s uint32 }

func (r *rnd) next() uint32 {
	r.s ^= r.s << 13
	r.s ^= r.s >> 17
	r.s ^= r.s << 5
	return r.s
}
func (r *rnd) f() float64               { return float64(r.next()) / 4294967296 }
func (r *rnd) n() float64               { return r.f()*2 - 1 }
func (r *rnd) rng(a, b float64) float64 { return a + (b-a)*r.f() }

// svf é o filtro de variável de estado: passa-baixa, passa-banda e passa-alta
// de uma vez, com o corte mudando a cada amostra.
type svf struct{ lo, bd float64 }

func (f *svf) step(x, cut, q float64) (lo, bd, hi float64) {
	k := 2 * math.Sin(math.Pi*math.Min(cut, sampleRate/6)/sampleRate)
	f.lo += k * f.bd
	hi = x - f.lo - q*f.bd
	f.bd += k * hi
	return f.lo, f.bd, hi
}

// env é o envelope: sobe em a segundos e cai exponencialmente com meia-vida d.
func env(t, a, d float64) float64 {
	if t < a {
		return t / a
	}
	return math.Exp2(-(t - a) / d)
}

func lerp(a, b, k float64) float64 { return a + (b-a)*k }

// add soma o som b em a a partir de off segundos, com ganho g.
func (a wave) add(b wave, off, g float64) {
	o := int(off * sampleRate)
	for i, v := range b {
		if j := o + i; j >= 0 && j < len(a) {
			a[j] += v * float32(g)
		}
	}
}

// norm deixa o pico em p.
func (a wave) norm(p float64) wave {
	m := float32(1e-6)
	for _, v := range a {
		m = max(m, float32(math.Abs(float64(v))))
	}
	k := float32(p) / m
	for i := range a {
		a[i] *= k
	}
	return a
}

// fadeEnds tira o estalo do começo e do fim.
func (a wave) fadeEnds() wave {
	n := min(len(a)/4, 220)
	for i := 0; i < n; i++ {
		k := float32(i) / float32(n)
		a[i] *= k
		a[len(a)-1-i] *= k
	}
	return a
}

// loopable cruza o fim com o começo para o laço não ter emenda.
func (a wave) loopable(cross float64) wave {
	n := int(cross * sampleRate)
	out := make(wave, len(a)-n)
	copy(out, a[:len(out)])
	for i := 0; i < n; i++ {
		k := float32(i) / float32(n)
		out[i] = a[len(out)+i]*(1-k) + a[i]*k
	}
	return out
}

// noiseBurst: ruído filtrado com corte que anda de c0 a c1.
func noiseBurst(r *rnd, dur, c0, c1, q, a, d float64, mode int) wave {
	w := secs(dur)
	var f svf
	for i := range w {
		t := float64(i) / sampleRate
		lo, bd, hi := f.step(r.n(), lerp(c0, c1, t/dur), q)
		v := [3]float64{lo, bd, hi}[mode]
		w[i] = float32(v * env(t, a, d))
	}
	return w
}

// sweep: seno (ou dente de serra, saw) com a frequência indo de f0 a f1.
func sweep(dur, f0, f1, a, d float64, saw bool) wave {
	w := secs(dur)
	ph := 0.0
	for i := range w {
		t := float64(i) / sampleRate
		ph += lerp(f0, f1, t/dur) / sampleRate
		v := math.Sin(ph * 2 * math.Pi)
		if saw {
			v = 2*(ph-math.Floor(ph)) - 1
		}
		w[i] = float32(v * env(t, a, d))
	}
	return w
}

// bell: parciais desafinadas, como sino ou cristal.
func bell(dur, f float64, parts []float64, d float64) wave {
	w := secs(dur)
	for i := range w {
		t := float64(i) / sampleRate
		v := 0.0
		for k, p := range parts {
			v += math.Sin(2*math.Pi*f*p*t) * math.Exp2(-t/(d/(1+float64(k)*0.5))) / (1 + float64(k))
		}
		w[i] = float32(v * env(t, 0.003, 10))
	}
	return w
}

// voice: dente de serra com vibrato e formante (passa-banda), para bichos.
func voice(r *rnd, dur float64, pitch func(t float64) float64, form, q, a, d, vib, rough float64) wave {
	w := secs(dur)
	var f, f2 svf
	ph := 0.0
	for i := range w {
		t := float64(i) / sampleRate
		p := pitch(t/dur) * (1 + 0.02*math.Sin(t*2*math.Pi*vib) + rough*r.n()*0.03)
		ph += p / sampleRate
		v := 2*(ph-math.Floor(ph)) - 1
		_, b1, _ := f.step(v, form, q)
		lo, _, _ := f2.step(v, form*0.5, 1)
		w[i] = float32((b1*0.8 + lo*0.4) * env(t, a, d) * (1 + rough*0.5*math.Sin(t*2*math.Pi*27)))
	}
	return w
}

// echo: eco simples (atraso com realimentação), para caverna e fantasma.
func (a wave) echo(delay, fb float64) wave {
	n := int(delay * sampleRate)
	out := make(wave, len(a)+n*4)
	copy(out, a)
	for i := n; i < len(out); i++ {
		out[i] += out[i-n] * float32(fb)
	}
	return out
}

// drive: saturação suave (deixa o som mais áspero).
func (a wave) drive(k float64) wave {
	for i, v := range a {
		a[i] = float32(math.Tanh(float64(v) * k))
	}
	return a
}

// genSounds gera todos os efeitos. Cada nome pode ter variações (o tocador
// sorteia uma), para o mesmo golpe não soar sempre igual.
func genSounds() map[string][]wave {
	r := &rnd{0x9e3779b9}
	s := map[string][]wave{}
	put := func(name string, ws ...wave) {
		for _, w := range ws {
			s[name] = append(s[name], w.fadeEnds().norm(0.85))
		}
	}
	vary := func(n int, f func(k float64) wave) []wave {
		out := make([]wave, n)
		for i := range out {
			out[i] = f(float64(i) / float64(max(1, n-1)))
		}
		return out
	}

	// --- combate ---
	put("swing", vary(3, func(k float64) wave {
		return noiseBurst(r, 0.22, 500+300*k, 2600+600*k, 0.7, 0.04, 0.05, 1)
	})...)
	put("swing_heavy", vary(2, func(k float64) wave {
		return noiseBurst(r, 0.34, 260+80*k, 1300, 0.6, 0.07, 0.08, 1)
	})...)
	put("hit", vary(3, func(k float64) wave {
		w := sweep(0.18, 150+30*k, 50, 0.002, 0.04, false)
		w.add(noiseBurst(r, 0.05, 3500, 900, 1, 0.001, 0.012, 0), 0, 0.8)
		return w
	})...)
	put("hit_bone", vary(3, func(k float64) wave {
		w := secs(0.2)
		for j := 0; j < 3; j++ {
			w.add(noiseBurst(r, 0.04, 2400+600*k, 2800, 0.12, 0.001, 0.006, 1), float64(j)*0.035+r.f()*0.01, 1-0.25*float64(j))
		}
		w.add(sweep(0.12, 220, 90, 0.002, 0.03, false), 0, 0.5)
		return w
	})...)
	put("hit_stone", vary(2, func(k float64) wave {
		w := noiseBurst(r, 0.3, 900, 200, 0.8, 0.001, 0.05, 0)
		w.add(sweep(0.25, 90+20*k, 40, 0.002, 0.06, false), 0, 0.8)
		return w
	})...)
	{
		w := sweep(0.2, 160, 50, 0.002, 0.05, false)
		w.add(noiseBurst(r, 0.05, 4000, 1200, 1, 0.001, 0.012, 0), 0, 0.9)
		w2 := secs(0.6)
		w2.add(w, 0, 1)
		w2.add(bell(0.55, 1800, []float64{1, 1.51, 2.17}, 0.09), 0.01, 0.5)
		put("crit", w2)
	}
	put("hurt", vary(2, func(k float64) wave {
		w := voice(r, 0.26, func(t float64) float64 { return 210 - 80*t + 30*k }, 700, 0.6, 0.01, 0.07, 6, 0.4)
		w.add(sweep(0.15, 120, 50, 0.002, 0.04, false), 0, 0.7)
		return w
	})...)
	{
		w := sweep(1.3, 110, 38, 0.02, 0.35, true)
		var f svf
		for i := range w {
			lo, _, _ := f.step(float64(w[i]), 500, 1)
			w[i] = float32(lo)
		}
		w.add(noiseBurst(r, 1.0, 600, 120, 0.8, 0.01, 0.25, 0), 0, 0.6)
		put("death", w)
	}
	put("block", bell(0.3, 1200, []float64{1, 2.4, 3.9}, 0.05))

	// --- magias ---
	put("fire", vary(2, func(k float64) wave {
		w := secs(0.7)
		var f svf
		for i := range w {
			t := float64(i) / sampleRate
			c := 400 + 1800*math.Sin(math.Min(1, t/0.6)*math.Pi)
			lo, _, _ := f.step(r.n(), c, 0.9)
			w[i] = float32(lo * env(t, 0.06, 0.18+0.05*k))
		}
		for j := 0; j < 18; j++ { // estalos
			w.add(noiseBurst(r, 0.012, 5000, 3000, 1, 0.0005, 0.002, 2), r.f()*0.6, 0.6*r.f())
		}
		return w
	})...)
	put("boom", vary(2, func(k float64) wave {
		w := noiseBurst(r, 1.4, 1400, 120, 0.9, 0.003, 0.22+0.08*k, 0)
		w.add(sweep(1.0, 70, 28, 0.003, 0.25, false), 0, 1)
		return w.drive(1.6)
	})...)
	put("ice", vary(2, func(k float64) wave {
		w := secs(0.6)
		for j := 0; j < 7; j++ {
			f := r.rng(2200, 5200)
			w.add(bell(0.3, f, []float64{1, 2.76}, 0.05), r.f()*0.25, 0.4+0.3*r.f())
		}
		w.add(noiseBurst(r, 0.4, 6000, 3000, 1, 0.005, 0.08, 2), 0, 0.3)
		return w
	})...)
	put("zap", vary(2, func(k float64) wave {
		w := secs(0.35)
		ph := 0.0
		for i := range w {
			t := float64(i) / sampleRate
			ph += (90 + 400*r.f()) / sampleRate
			v := math.Copysign(1, math.Sin(ph*2*math.Pi)) * 0.5
			w[i] = float32(v * env(t, 0.002, 0.08))
		}
		w.add(noiseBurst(r, 0.3, 7000, 2500, 0.5, 0.001, 0.05, 2), 0, 0.8)
		return w
	})...)
	{
		w := secs(3.2)
		w.add(noiseBurst(r, 0.25, 8000, 2000, 0.6, 0.001, 0.04, 2), 0, 1)
		rum := secs(3.0)
		var f svf
		for i := range rum {
			t := float64(i) / sampleRate
			lo, _, _ := f.step(r.n(), 180+120*math.Sin(t*9), 0.9)
			rum[i] = float32(lo * env(t, 0.05, 0.7) * (0.7 + 0.3*math.Sin(t*13+math.Sin(t*5)*3)))
		}
		w.add(rum, 0.05, 3)
		put("thunder", w.drive(1.4))
	}
	{
		w := secs(1.0)
		for j, n := range []float64{523, 659, 784, 1046} {
			w.add(bell(0.6, n, []float64{1, 2, 3}, 0.12), float64(j)*0.08, 0.6)
		}
		put("heal", w)
	}
	{
		w := secs(1.6)
		for _, n := range []float64{880, 1108, 1318, 1760} {
			w.add(bell(1.5, n, []float64{1, 2.01, 3.03}, 0.35), 0, 0.4)
		}
		put("holy", w)
	}
	{
		w := secs(1.1)
		for _, f0 := range []float64{55, 56.3, 82.5} {
			sw := sweep(1.1, f0, f0*0.9, 0.35, 0.3, true)
			var f svf
			for i := range sw {
				lo, _, _ := f.step(float64(sw[i]), 400, 1)
				sw[i] = float32(lo)
			}
			w.add(sw, 0, 0.6)
		}
		w.add(noiseBurst(r, 1.0, 300, 900, 0.6, 0.3, 0.2, 1), 0, 0.5)
		put("dark", w)
	}
	put("arrow", vary(2, func(k float64) wave {
		w := secs(0.35)
		// corda: Karplus-Strong curtinho
		n := int(sampleRate / (170 + 30*k))
		buf := make([]float64, n)
		for i := range buf {
			buf[i] = r.n()
		}
		for i := 0; i < int(0.2*sampleRate); i++ {
			j := i % n
			v := buf[j]
			buf[j] = 0.5 * (buf[j] + buf[(j+1)%n]) * 0.985
			w[i] = float32(v * 0.6)
		}
		w.add(noiseBurst(r, 0.3, 3000, 5000, 0.2, 0.05, 0.06, 1), 0.02, 0.5)
		return w
	})...)
	put("missile", sweep(0.3, 420, 1500, 0.01, 0.08, false))
	put("blink", sweep(0.22, 2200, 280, 0.003, 0.05, false))
	put("buff", vary(1, func(float64) wave {
		w := sweep(0.6, 200, 600, 0.15, 0.15, true)
		var f svf
		for i := range w {
			lo, _, _ := f.step(float64(w[i]), 1200, 1)
			w[i] = float32(lo)
		}
		w.add(bell(0.5, 1320, []float64{1, 1.5}, 0.15), 0.2, 0.4)
		return w
	})...)
	put("puff", noiseBurst(r, 0.5, 800, 300, 0.8, 0.05, 0.1, 0))
	put("warn", vary(1, func(float64) wave {
		w := noiseBurst(r, 0.8, 200, 120, 0.7, 0.2, 0.25, 0)
		w.add(sweep(0.8, 60, 50, 0.2, 0.25, false), 0, 0.8)
		return w
	})...)

	// --- coisas quebrando ---
	put("wood_break", vary(2, func(k float64) wave {
		w := secs(0.9)
		w.add(noiseBurst(r, 0.06, 6000, 2000, 0.5, 0.0005, 0.01, 1), 0, 1)
		for j := 0; j < 14; j++ { // fibras rangendo
			w.add(noiseBurst(r, 0.05, r.rng(500, 1400), 500, 0.25, 0.002, 0.012, 1), 0.04+r.f()*0.45, 0.5*r.f())
		}
		w.add(sweep(0.4, 110+20*k, 50, 0.002, 0.08, false), 0.02, 0.7)
		return w
	})...)
	{
		w := secs(2.0) // árvore caindo: estala, range e bate no chão
		w.add(s["wood_break"][0], 0, 1)
		for j := 0; j < 30; j++ {
			w.add(noiseBurst(r, 0.06, r.rng(300, 900), 400, 0.3, 0.002, 0.015, 1), 0.3+float64(j)*0.03, 0.4)
		}
		w.add(noiseBurst(r, 0.8, 800, 100, 0.9, 0.002, 0.15, 0), 1.25, 1.3)
		w.add(sweep(0.6, 70, 30, 0.002, 0.15, false), 1.25, 1)
		put("tree_fall", w)
	}
	put("stone_break", vary(2, func(k float64) wave {
		w := secs(0.8)
		for j := 0; j < 26; j++ {
			w.add(noiseBurst(r, 0.04, r.rng(600, 2200), 300, 0.4, 0.001, 0.008, 1), r.f()*0.5, r.f())
		}
		w.add(noiseBurst(r, 0.4, 900, 150, 0.9, 0.001, 0.06, 0), 0, 1)
		w.add(sweep(0.3, 80, 35, 0.002, 0.07, false), 0, 0.8)
		return w
	})...)
	{
		w := secs(1.0)
		for j := 0; j < 12; j++ {
			w.add(bell(0.5, r.rng(2500, 6500), []float64{1, 2.3}, 0.06), r.f()*0.35, 0.3+0.4*r.f())
		}
		w.add(noiseBurst(r, 0.3, 9000, 4000, 0.6, 0.001, 0.05, 2), 0, 0.7)
		put("crystal_break", w)
	}
	put("grass", vary(3, func(k float64) wave {
		return noiseBurst(r, 0.16, 2500+1000*k, 6000, 0.6, 0.02, 0.03, 2)
	})...)
	put("door", vary(1, func(float64) wave {
		w := noiseBurst(r, 1.6, 260, 140, 0.8, 0.2, 0.4, 0)
		for j := 0; j < 10; j++ {
			w.add(noiseBurst(r, 0.05, 700, 400, 0.4, 0.002, 0.01, 1), r.f()*1.2, 0.5)
		}
		return w
	})...)

	// --- mundo e interface ---
	put("coin", vary(2, func(k float64) wave {
		w := secs(0.3)
		w.add(bell(0.2, 1900+200*k, []float64{1, 2.7}, 0.04), 0, 1)
		w.add(bell(0.2, 2500+200*k, []float64{1, 2.7}, 0.04), 0.06, 0.8)
		return w
	})...)
	put("pickup", sweep(0.12, 600, 1200, 0.005, 0.03, false))
	{
		w := secs(1.4)
		for j, n := range []float64{392, 523, 659, 784, 1046} {
			w.add(bell(0.9, n, []float64{1, 2, 3}, 0.2), float64(j)*0.1, 0.5)
		}
		w.add(noiseBurst(r, 1.0, 6000, 9000, 0.5, 0.3, 0.2, 2), 0.3, 0.12)
		put("levelup", w)
	}
	put("click", noiseBurst(r, 0.02, 3500, 3000, 0.3, 0.0005, 0.004, 1))
	put("msg", bell(0.15, 1400, []float64{1, 2}, 0.03))
	{
		w := secs(1.4)
		w.add(noiseBurst(r, 1.0, 300, 2500, 0.6, 0.15, 0.25, 0), 0, 1)
		for j := 0; j < 20; j++ {
			w.add(noiseBurst(r, 0.012, 5000, 3000, 1, 0.0005, 0.002, 2), 0.2+r.f()*1.0, 0.5*r.f())
		}
		put("bonfire", w)
	}
	put("step_grass", vary(4, func(k float64) wave { return noiseBurst(r, 0.07, 1800+900*k, 3000, 0.7, 0.004, 0.015, 1) })...)
	put("step_stone", vary(4, func(k float64) wave {
		w := noiseBurst(r, 0.05, 900+300*k, 600, 0.5, 0.001, 0.008, 1)
		w.add(sweep(0.05, 140, 90, 0.001, 0.01, false), 0, 0.4)
		return w
	})...)
	put("step_mud", vary(3, func(k float64) wave { return noiseBurst(r, 0.12, 300+100*k, 700, 0.3, 0.01, 0.03, 1) })...)
	put("step_snow", vary(3, func(k float64) wave { return noiseBurst(r, 0.1, 3000+800*k, 1500, 0.5, 0.02, 0.025, 1) })...)
	put("splash", vary(2, func(k float64) wave {
		w := noiseBurst(r, 0.35, 1500, 600, 0.6, 0.01, 0.07, 1)
		for j := 0; j < 4; j++ {
			w.add(sweep(0.06, r.rng(400, 900), r.rng(1200, 2000), 0.002, 0.015, false), r.f()*0.15, 0.3)
		}
		return w
	})...)

	// --- bichos ---
	put("wolf", vary(2, func(k float64) wave {
		return voice(r, 1.9, func(t float64) float64 {
			return 380 + 320*math.Sin(math.Min(1, t*1.6)*math.Pi*0.5) - 160*math.Max(0, t-0.7)/0.3 + 40*k
		}, 900, 0.5, 0.25, 0.5, 5, 0)
	})...)
	put("growl", vary(3, func(k float64) wave {
		return voice(r, 0.7, func(t float64) float64 { return 85 + 25*k + 15*math.Sin(t*9) }, 450, 0.8, 0.05, 0.2, 3, 1).drive(1.8)
	})...)
	put("rat", vary(2, func(k float64) wave {
		w := secs(0.4)
		for j := 0; j < 3; j++ {
			w.add(sweep(0.06, 3000+500*k, 4200, 0.004, 0.015, false), float64(j)*0.1+r.f()*0.03, 0.8)
		}
		return w
	})...)
	put("bat", vary(2, func(k float64) wave {
		w := secs(0.35)
		for j := 0; j < 5; j++ {
			w.add(sweep(0.025, 7500, 5500+600*k, 0.001, 0.006, false), float64(j)*0.05, 0.7)
		}
		return w
	})...)
	put("bones", vary(2, func(k float64) wave {
		w := secs(0.6)
		for j := 0; j < 14; j++ {
			w.add(noiseBurst(r, 0.03, r.rng(1800, 3600), 2500, 0.12, 0.001, 0.005, 1), r.f()*0.5, 0.5+0.5*r.f())
		}
		return w
	})...)
	put("insect", vary(2, func(k float64) wave {
		w := secs(0.7)
		for j := 0; j < 22; j++ {
			w.add(noiseBurst(r, 0.015, 4200+800*k, 4000, 0.15, 0.001, 0.003, 1), float64(j)*0.028, 0.5+0.5*math.Sin(float64(j)*0.4))
		}
		return w
	})...)
	put("stone_grind", vary(1, func(float64) wave {
		w := secs(0.9)
		var f svf
		for i := range w {
			t := float64(i) / sampleRate
			lo, _, _ := f.step(r.n()*(0.5+0.5*math.Abs(math.Sin(t*40))), 320, 0.7)
			w[i] = float32(lo * env(t, 0.1, 0.25))
		}
		return w
	})...)
	put("demon", vary(2, func(k float64) wave {
		w := voice(r, 1.5, func(t float64) float64 { return 62 + 20*math.Sin(t*math.Pi) - 10*k }, 380, 0.9, 0.1, 0.4, 4, 1.5)
		w.add(noiseBurst(r, 1.4, 500, 250, 0.8, 0.15, 0.35, 0), 0, 0.6)
		return w.drive(2.4)
	})...)
	put("ghost", vary(2, func(k float64) wave {
		w := secs(1.6)
		ph := 0.0
		for i := range w {
			t := float64(i) / sampleRate
			ph += (300 + 220*math.Sin(t*1.7+k) + 12*math.Sin(t*2*math.Pi*6)) / sampleRate
			w[i] = float32(math.Sin(ph*2*math.Pi) * env(t, 0.4, 0.4) * 0.6)
		}
		return w.echo(0.18, 0.45)
	})...)
	put("gurgle", vary(2, func(k float64) wave {
		w := secs(0.7)
		for j := 0; j < 7; j++ {
			w.add(sweep(0.07, r.rng(250, 500), r.rng(700, 1100), 0.003, 0.02, false), r.f()*0.55, 0.6)
		}
		return w
	})...)
	put("moan", vary(2, func(k float64) wave {
		return voice(r, 1.3, func(t float64) float64 { return 115 - 35*t + 10*k }, 550, 0.7, 0.2, 0.35, 3, 0.5)
	})...)
	put("horse", vary(2, func(k float64) wave {
		return voice(r, 1.1, func(t float64) float64 {
			return 520 + 380*math.Sin(math.Min(1, t*2.2)*math.Pi) - 200*t + 30*k
		}, 1400, 0.4, 0.03, 0.25, 11, 0.6)
	})...)
	put("grunt", vary(2, func(k float64) wave {
		w := secs(0.5)
		for j := 0; j < 2; j++ {
			w.add(voice(r, 0.14, func(t float64) float64 { return 140 + 30*k - 40*t }, 500, 0.8, 0.01, 0.04, 2, 1), float64(j)*0.2, 1)
		}
		return w
	})...)
	put("roar", vary(2, func(k float64) wave {
		w := voice(r, 2.4, func(t float64) float64 { return 95 - 40*t + 25*math.Sin(t*math.Pi) + 10*k }, 600, 0.8, 0.2, 0.6, 5, 1.6)
		w.add(noiseBurst(r, 2.2, 1200, 300, 0.7, 0.2, 0.5, 0), 0, 0.8)
		return w.drive(2.2)
	})...)
	put("bird", vary(4, func(k float64) wave {
		w := secs(0.5)
		base := 2600 + 1600*k
		for j := 0; j < 2+int(k*3); j++ {
			w.add(sweep(0.07, base, base*1.35, 0.004, 0.02, false), float64(j)*0.09, 0.7)
		}
		return w
	})...)
	put("owl", vary(1, func(float64) wave {
		w := secs(1.3)
		w.add(sweep(0.35, 400, 380, 0.05, 0.12, false), 0, 1)
		w.add(sweep(0.5, 390, 360, 0.05, 0.18, false), 0.5, 1)
		return w
	})...)
	put("frog", vary(2, func(k float64) wave {
		w := secs(0.5)
		for j := 0; j < 4; j++ {
			w.add(voice(r, 0.06, func(float64) float64 { return 170 + 30*k }, 600, 0.6, 0.005, 0.02, 1, 0.5), float64(j)*0.07, 1)
		}
		return w
	})...)
	put("drip", vary(3, func(k float64) wave {
		return sweep(0.12, 900+400*k, 2300+300*k, 0.001, 0.02, false).echo(0.13, 0.35)
	})...)
	put("bubble", vary(3, func(k float64) wave { return sweep(0.16, 110+40*k, 300, 0.005, 0.04, false) })...)
	genVoices(s)
	return s
}

// genLoops gera os laços de ambiente (já sem emenda).
func genLoops() map[string]wave {
	r := &rnd{0x1234567}
	L := map[string]wave{}

	{ // vento: ruído grave com o corte e o volume subindo e descendo devagar
		w := secs(13)
		var f svf
		for i := range w {
			t := float64(i) / sampleRate
			m := 0.5 + 0.3*math.Sin(t*0.9) + 0.2*math.Sin(t*2.3+1)
			lo, bd, _ := f.step(r.n(), 220+480*m, 0.5)
			w[i] = float32((lo*0.7 + bd*0.5) * (0.45 + 0.55*m))
		}
		L["wind"] = w.loopable(1).norm(0.6)
	}
	{ // grilos: vários, cada um no seu ritmo
		w := secs(9)
		for c := 0; c < 5; c++ {
			f0 := r.rng(3900, 5200)
			rate := r.rng(2.2, 4)
			off := r.f()
			for t := off; t < 8.6; t += 1 / rate {
				if r.f() < 0.25 {
					continue
				}
				ch := secs(0.12)
				for i := range ch {
					tt := float64(i) / sampleRate
					ch[i] = float32(math.Sin(2*math.Pi*f0*tt) * (0.5 + 0.5*math.Sin(2*math.Pi*55*tt)) * env(tt, 0.005, 0.03))
				}
				w.add(ch, t, 0.25)
			}
		}
		L["crickets"] = w.loopable(0.4).norm(0.45)
	}
	{ // chuva: chiado e pingos
		w := secs(7)
		var f, f2 svf
		for i := range w {
			_, _, hi := f.step(r.n(), 3000, 0.7)
			lo, _, _ := f2.step(r.n(), 900, 0.7)
			w[i] = float32(hi*0.4 + lo*0.5)
		}
		for j := 0; j < 220; j++ {
			w.add(noiseBurst(r, 0.02, r.rng(2000, 6000), 3000, 0.3, 0.0005, 0.003, 1), r.f()*6.9, 0.3*r.f())
		}
		L["rain"] = w.loopable(0.5).norm(0.6)
	}
	{ // mar e lago: ondas indo e vindo
		w := secs(12)
		var f svf
		for i := range w {
			t := float64(i) / sampleRate
			wave := math.Pow(0.5+0.5*math.Sin(t*2*math.Pi/6), 2)
			lo, _, _ := f.step(r.n(), 300+900*wave, 0.6)
			w[i] = float32(lo * (0.25 + 0.75*wave))
		}
		L["sea"] = w.loopable(1).norm(0.55)
	}
	{ // caverna: um zumbido de sala grave
		w := secs(10)
		var f svf
		for i := range w {
			t := float64(i) / sampleRate
			lo, _, _ := f.step(r.n(), 110+30*math.Sin(t*0.6), 0.6)
			w[i] = float32(lo * (0.8 + 0.2*math.Sin(t*0.4)))
		}
		L["cave"] = w.loopable(1).norm(0.5)
	}
	{ // lava: ronco e bolhas
		w := secs(8)
		var f svf
		for i := range w {
			lo, _, _ := f.step(r.n(), 90, 0.7)
			w[i] = float32(lo)
		}
		w = w.norm(0.6)
		for j := 0; j < 22; j++ {
			w.add(sweep(0.18, r.rng(90, 160), r.rng(220, 380), 0.005, 0.05, false), r.f()*7.6, 0.4)
		}
		L["lava"] = w.loopable(0.4).norm(0.6)
	}
	{ // fogo de fogueira: estalos e chiado
		w := secs(5)
		var f svf
		for i := range w {
			lo, _, _ := f.step(r.n(), 700, 0.6)
			w[i] = float32(lo * 0.25)
		}
		for j := 0; j < 120; j++ {
			w.add(noiseBurst(r, 0.015, r.rng(2500, 7000), 3000, 0.6, 0.0005, 0.002+0.004*r.f(), 2), r.f()*4.9, r.f())
		}
		L["fire"] = w.loopable(0.3).norm(0.5)
	}
	return L
}

// Formantes das vogais (F1, F2, F3 em Hz), para vozes que lembram fala.
var vowelF = map[byte][3]float64{
	'a': {800, 1200, 2500}, 'e': {500, 1750, 2500}, 'i': {320, 2250, 3000},
	'o': {520, 900, 2400}, 'u': {360, 700, 2300}, 'y': {420, 1400, 2300}, // y: vogal neutra (schwa)
}

// formantVoice: pulso glotal (dente de serra) passando por três passa-bandas
// que andam pelas vogais de vowels ao longo do som. pitch(t) dá a altura,
// rough faz a voz áspera (orc, troll), breath mistura sopro (múmia, espectro).
func formantVoice(r *rnd, dur float64, pitch func(t float64) float64, vowels string, rough, breath, a, d float64) wave {
	w := secs(dur)
	var f1, f2, f3 svf
	ph := 0.0
	n := len(vowels)
	for i := range w {
		t := float64(i) / sampleRate
		u := t / dur
		// vogal atual e a próxima, misturadas
		pos := u * float64(n-1)
		k := int(pos)
		fr := pos - float64(k)
		va, vb := vowelF[vowels[min(k, n-1)]], vowelF[vowels[min(k+1, n-1)]]
		p := pitch(u) * (1 + 0.012*math.Sin(t*2*math.Pi*5.5) + rough*0.04*r.n())
		ph += p / sampleRate
		src := 2*(ph-math.Floor(ph)) - 1
		src = src*(1-breath) + r.n()*breath*1.4
		if rough > 0 {
			src *= 1 + rough*0.6*math.Sin(t*2*math.Pi*31) // estalo de garganta
		}
		_, b1, _ := f1.step(src, lerp(va[0], vb[0], fr), 0.18)
		_, b2, _ := f2.step(src, lerp(va[1], vb[1], fr), 0.14)
		_, b3, _ := f3.step(src, lerp(va[2], vb[2], fr), 0.12)
		w[i] = float32((b1 + 0.7*b2 + 0.35*b3) * env(t, a, d))
	}
	return w
}

// genVoices são as vozes próprias de cada inimigo (somam-se a genSounds).
func genVoices(s map[string][]wave) {
	r := &rnd{0x51ed270b}
	put := func(name string, ws ...wave) {
		for _, w := range ws {
			s[name] = append(s[name], w.fadeEnds().norm(0.85))
		}
	}
	// Orc: grito grosso "ruh-GAH", áspero.
	for k := 0.0; k < 3; k++ {
		put("orc", formantVoice(r, 0.75, func(t float64) float64 { return 105 + 40*math.Sin(t*math.Pi) - 10*k }, "uoa", 0.9, 0.05, 0.03, 0.22).drive(2))
	}
	// Troll e gigante: urro lento e muito grave.
	put("troll", formantVoice(r, 1.3, func(t float64) float64 { return 62 + 18*math.Sin(t*math.Pi) }, "ooa", 1.2, 0.1, 0.12, 0.4).drive(2.2))
	put("giant", formantVoice(r, 1.8, func(t float64) float64 { return 48 + 14*math.Sin(t*math.Pi) }, "uoao", 1, 0.15, 0.2, 0.6).drive(2))
	// Yeti: bufa e ruge, com muito sopro.
	for k := 0.0; k < 2; k++ {
		w := formantVoice(r, 1.0, func(t float64) float64 { return 140 - 50*t + 10*k }, "aao", 0.7, 0.45, 0.08, 0.3)
		w.add(noiseBurst(r, 0.3, 900, 400, 0.5, 0.01, 0.06, 0), 0, 0.6)
		put("yeti", w.drive(1.6))
	}
	// Esqueleto: ossos batendo e a mandíbula estalando.
	for k := 0.0; k < 3; k++ {
		w := secs(0.55)
		for j := 0; j < 10; j++ {
			w.add(noiseBurst(r, 0.025, r.rng(1800, 3800), 2600, 0.1, 0.0008, 0.004, 1), r.f()*0.45, 0.4+0.6*r.f())
		}
		for j := 0; j < 2; j++ { // mandíbula: dois estalos secos e graves
			w.add(noiseBurst(r, 0.04, 900+200*k, 600, 0.15, 0.001, 0.008, 1), 0.1+float64(j)*0.13, 1)
		}
		put("skeleton", w)
	}
	// Aranha e escorpião: chiado e patas.
	for k := 0.0; k < 2; k++ {
		w := noiseBurst(r, 0.6, 5500, 4000, 0.4, 0.05, 0.15, 2)
		for j := 0; j < 16; j++ {
			w.add(noiseBurst(r, 0.012, 3800+600*k, 3600, 0.15, 0.0005, 0.003, 1), float64(j)*0.035, 0.6)
		}
		put("spider", w)
		w2 := secs(0.5)
		for j := 0; j < 6; j++ {
			w2.add(noiseBurst(r, 0.02, 2500, 2400, 0.08, 0.0005, 0.004, 1), float64(j)*0.06+r.f()*0.02, 1)
		}
		w2.add(noiseBurst(r, 0.3, 6000, 4500, 0.5, 0.02, 0.07, 2), 0.2, 0.5)
		put("scorpion", w2)
	}
	// Escaravelho: estalos secos e rápidos.
	{
		w := secs(0.4)
		for j := 0; j < 12; j++ {
			w.add(noiseBurst(r, 0.01, 3000, 2800, 0.1, 0.0004, 0.002, 1), float64(j)*0.03, 0.8)
		}
		put("scarab", w)
	}
	// Asura: canto em vogais, quase melodia.
	for k := 0.0; k < 2; k++ {
		notes := []float64{220, 247, 196, 220}
		put("asura", formantVoice(r, 1.4, func(t float64) float64 { return notes[min(3, int(t*4))] * (1 + 0.05*k) }, "aeia", 0, 0.05, 0.1, 0.5))
	}
	// Tritão: gorgolejo com coaxar.
	{
		w := formantVoice(r, 0.7, func(t float64) float64 { return 160 - 40*t }, "uou", 0.6, 0.2, 0.02, 0.2)
		for j := 0; j < 5; j++ {
			w.add(sweep(0.06, r.rng(300, 500), r.rng(800, 1100), 0.002, 0.015, false), r.f()*0.55, 0.4)
		}
		put("triton", w)
	}
	// Golem: pedra rangendo e um ronco fundo.
	{
		w := secs(1.0)
		w.add(s["stone_grind"][0], 0, 1)
		w.add(sweep(0.9, 50, 38, 0.1, 0.3, true), 0, 0.6)
		put("golem", w.drive(1.4))
	}
	// Apóstolo: rosnado molhado e gutural, que sobe num grito.
	for k := 0.0; k < 2; k++ {
		w := formantVoice(r, 1.2, func(t float64) float64 { return 90 + 120*t*t - 10*k }, "uoaa", 1.3, 0.3, 0.05, 0.35)
		w.add(noiseBurst(r, 1.0, 700, 1600, 0.5, 0.1, 0.3, 1), 0, 0.4)
		put("apostle", w.drive(2.3))
	}
	// Anão guardião: "hm!" grave e o aço batendo.
	{
		w := formantVoice(r, 0.4, func(t float64) float64 { return 120 - 20*t }, "yu", 0.4, 0.05, 0.01, 0.1)
		w2 := secs(0.7)
		w2.add(w, 0, 1)
		w2.add(bell(0.5, 1250, []float64{1, 2.4, 3.9}, 0.07), 0.25, 0.6)
		put("dwarf", w2)
	}
	// Múmia: gemido com sopro.
	for k := 0.0; k < 2; k++ {
		put("mummy", formantVoice(r, 1.4, func(t float64) float64 { return 110 - 30*t + 8*k }, "ouu", 0.3, 0.55, 0.25, 0.4))
	}
	// Espectro gélido: lamento agudo com sussurro.
	{
		w := formantVoice(r, 1.6, func(t float64) float64 { return 420 + 160*math.Sin(t*math.Pi*1.5) }, "iuo", 0, 0.5, 0.4, 0.45)
		put("wraith", w.echo(0.2, 0.4))
	}
	// Rastejante e casca de fungo: arrasto e estouro úmido.
	{
		w := secs(0.6)
		for j := 0; j < 8; j++ {
			w.add(noiseBurst(r, 0.06, r.rng(600, 1200), 500, 0.3, 0.004, 0.015, 1), r.f()*0.5, 0.7)
		}
		put("crawler", w)
		w2 := secs(0.6)
		for j := 0; j < 4; j++ {
			w2.add(noiseBurst(r, 0.08, 500, 200, 0.6, 0.002, 0.02, 0), r.f()*0.4, 1)
			w2.add(sweep(0.05, 200, 90, 0.001, 0.01, false), r.f()*0.4, 0.6)
		}
		put("fungus", w2)
	}
	// Horror das raízes: madeira rangendo e um gemido fundo.
	{
		w := secs(1.2)
		for j := 0; j < 20; j++ {
			w.add(noiseBurst(r, 0.05, r.rng(400, 900), 400, 0.25, 0.002, 0.012, 1), 0.05+float64(j)*0.04, 0.4)
		}
		w.add(formantVoice(r, 1.0, func(t float64) float64 { return 75 }, "ou", 0.5, 0.3, 0.2, 0.3), 0.2, 0.8)
		put("roots", w)
	}
	// Arconte e demônios ganham um eco de catedral.
	put("archon", formantVoice(r, 1.5, func(t float64) float64 { return 130 + 20*math.Sin(t*6) }, "aoa", 0.2, 0.1, 0.2, 0.5).echo(0.25, 0.45))
}
