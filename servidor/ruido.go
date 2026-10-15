package main

import (
	"math"
	"runtime"
	"sync"
)

func hash2(x, y int, seed uint32) uint32 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + seed*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	return h ^ (h >> 16)
}

func rnd01(x, y int, seed uint32) float64 {
	return float64(hash2(x, y, seed)&0xffff) / 65535
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

func valueNoise(x, y float64, seed uint32) float64 {
	x0, y0 := math.Floor(x), math.Floor(y)
	fx, fy := x-x0, y-y0
	fx, fy = fx*fx*(3-2*fx), fy*fy*(3-2*fy)
	ix, iy := int(x0), int(y0)
	top := lerp(rnd01(ix, iy, seed), rnd01(ix+1, iy, seed), fx)
	bot := lerp(rnd01(ix, iy+1, seed), rnd01(ix+1, iy+1, seed), fx)
	return lerp(top, bot, fy)
}

func fbm(x, y float64, seed uint32) float64 {
	return valueNoise(x, y, seed)*0.65 + valueNoise(x*2.7, y*2.7, seed+1)*0.35
}

func dist(ax, ay, bx, by float64) float64 {
	return math.Hypot(ax-bx, ay-by)
}

func pick(set string, h uint32) byte { return set[h%uint32(len(set))] }

// parallelRows roda fn para cada linha de 0 a n-1, dividindo entre os núcleos.
func parallelRows(n int, fn func(y int)) {
	workers := runtime.NumCPU()
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for y := w; y < n; y += workers {
				fn(y)
			}
		}(w)
	}
	wg.Wait()
}

func mix(a, b rgb, t float64) rgb {
	ch := func(sh uint) rgb {
		return rgb(lerp(float64(a>>sh&0xff), float64(b>>sh&0xff), t))
	}
	return ch(16)<<16 | ch(8)<<8 | ch(0)
}
