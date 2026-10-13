package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

//go:embed assets/art.json
var assets embed.FS

type rgb [3]int
type point struct{ x, y float64 }
type stop struct {
	offset float64
	color  rgb
}

func (s *stop) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if len(raw) != 2 {
		return fmt.Errorf("invalid gradient stop")
	}
	if err := json.Unmarshal(raw[0], &s.offset); err != nil {
		return err
	}
	return json.Unmarshal(raw[1], &s.color)
}

type face struct {
	Points [][2]float64 `json:"points"`
	Axis   [4]float64   `json:"axis"`
	Stops  []stop       `json:"stops"`
}
type lettering struct {
	Rows, Cols int
	Bits       []string
}
type artwork struct {
	Shapes  []face
	Words   map[string]lettering `json:"word_sizes"`
	Octants string
}

var brandLabel = "Codex"
var art artwork
var glyphs []rune
var cellRatio = 2.12
var ink = rgb{238, 233, 228}
var muted = rgb{128, 141, 155}
var bg = rgb{14, 15, 18}
var cyan = rgb{61, 206, 242}

func loadArt() error {
	raw, err := assets.ReadFile("assets/art.json")
	if err != nil {
		return err
	}
	if err = json.Unmarshal(raw, &art); err != nil {
		return err
	}
	if len(art.Shapes) == 0 || len(art.Words) == 0 {
		return fmt.Errorf("empty embedded artwork")
	}
	glyphs = []rune(art.Octants)
	mode := os.Getenv("PANGU_GLYPHS")
	if mode != "octant" && !(mode == "" && strings.Contains(strings.ToLower(os.Getenv("TERM_PROGRAM")), "ghostty")) {
		quads := []rune(" ▘▝▀▖▌▞▛▗▚▐▜▄▙▟█")
		glyphs = make([]rune, 256)
		for bits := 0; bits < 256; bits++ {
			q := 0
			for i, p := range [][2]int{{0, 2}, {1, 3}, {4, 6}, {5, 7}} {
				if bits&(1<<p[0]|1<<p[1]) != 0 {
					q |= 1 << i
				}
			}
			glyphs[bits] = quads[q]
		}
	}
	if v, e := strconv.ParseFloat(os.Getenv("PANGU_CELL_RATIO"), 64); e == nil && v >= 1 && v <= 4 {
		cellRatio = v
	}
	if os.Getenv("PANGU_THEME") == "light" {
		ink = rgb{30, 26, 27}
		muted = rgb{85, 100, 118}
		bg = rgb{250, 250, 250}
	}
	return nil
}
func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
func mix(a, b rgb, t float64) rgb {
	t = clamp(t, 0, 1)
	var c rgb
	for i := range c {
		c[i] = int(math.Round(float64(a[i]) + (float64(b[i]-a[i]))*t))
	}
	return c
}
func ease(t float64) float64 { t = clamp(t, 0, 1); return 1 - math.Pow(1-t, 3) }
func inside(poly [][2]float64, x, y float64) bool {
	yes := false
	j := len(poly) - 1
	for i := range poly {
		a, b := poly[i], poly[j]
		if (a[1] > y) != (b[1] > y) && x < (b[0]-a[0])*(y-a[1])/(b[1]-a[1])+a[0] {
			yes = !yes
		}
		j = i
	}
	return yes
}
func faceColor(f face, x, y float64) rgb {
	dx, dy := f.Axis[2]-f.Axis[0], f.Axis[3]-f.Axis[1]
	t := clamp(((x-f.Axis[0])*dx+(y-f.Axis[1])*dy)/(dx*dx+dy*dy), 0, 1)
	for i := 1; i < len(f.Stops); i++ {
		a, b := f.Stops[i-1], f.Stops[i]
		if t <= b.offset {
			return mix(a.color, b.color, (t-a.offset)/(b.offset-a.offset))
		}
	}
	return f.Stops[len(f.Stops)-1].color
}
func charWidth(r rune) int {
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || r >= 0x200b && r <= 0x200f || r >= 0xfe00 && r <= 0xfe0f {
		return 0
	}
	if r >= 0x1100 && (r <= 0x115f || r >= 0x2e80 && r <= 0xa4cf || r >= 0xac00 && r <= 0xd7a3 || r >= 0xf900 && r <= 0xfaff || r >= 0xfe10 && r <= 0xfe6f || r >= 0xff00 && r <= 0xff60 || r >= 0xffe0 && r <= 0xffe6 || r >= 0x1f300 && r <= 0x1faff || r >= 0x20000 && r <= 0x3fffd) {
		return 2
	}
	return 1
}
func textWidth(s string) int {
	n := 0
	for _, r := range s {
		n += charWidth(r)
	}
	return n
}

type position struct{ r, c int }
type pixel struct {
	ch    rune
	color rgb
}
type canvas map[position]pixel
type layout struct{ cols, rows, wordRows, wordCols, logoCols, logoRows, width, height, wordTop, sloganTop, tagTop, platformTop int }

func newLayout(cols, wr int) layout {
	l := layout{cols: cols, wordRows: wr}
	if wr > 0 {
		l.wordCols = art.Words[strconv.Itoa(wr)].Cols
		l.logoCols = max(10, wr*10)
		l.logoRows = int(math.Ceil(49 * float64(l.logoCols) * 2 / 71 / (cellRatio / 2) / 4))
		l.wordTop = l.logoRows + max(1, int(math.Round(float64(l.logoRows)*.12)))
		gap := 1
		if wr >= 3 {
			gap = 2
		}
		l.sloganTop = l.wordTop + wr + gap
	}
	l.tagTop = l.sloganTop + 1
	l.platformTop = l.tagTop + 2
	if wr >= 3 {
		l.platformTop++
	}
	l.height = l.platformTop + 1
	l.width = max(l.logoCols, l.wordCols, textWidth("──  词元无限 · Codex  ──"), len("T O K E N S   I N F I N I T Y"))
	return l
}
func choose(cols, rows int) layout {
	for wr := 6; wr >= 1; wr-- {
		l := newLayout(cols, wr)
		if l.width <= cols-4 && l.height <= rows {
			return l
		}
	}
	return newLayout(cols, 0)
}
func (l layout) center(n int) int { return max(0, (l.cols-n)/2) }
func (c canvas) text(row, col int, text string, color rgb) {
	for _, r := range text {
		c[position{row, col}] = pixel{r, color}
		col += charWidth(r)
	}
}
func (l layout) logo(out canvas, top int, t float64) {
	if l.wordRows == 0 {
		return
	}
	scale := float64(l.logoCols) * 2 / 71
	yscale := scale / (cellRatio / 2)
	spare := float64(l.logoRows*4) - 49*yscale
	xoff := l.center(l.logoCols)
	for r := 0; r < l.logoRows; r++ {
		for col := 0; col < l.logoCols; col++ {
			bits := 0
			total := rgb{}
			hits := 0
			for sub := 0; sub < 8; sub++ {
				x := (float64(col*2+sub%2) + .5) / scale
				y := (float64(r*4+sub/2) + .5 - spare/2) / yscale
				for index, f := range art.Shapes {
					sx, sy := x, y
					if t < 1.62 {
						raw := (t - .30 - .055*float64(index)) / .90
						if raw <= 0 {
							continue
						}
						p := ease(raw)
						var cx, cy float64
						for _, v := range f.Points {
							cx += v[0]
							cy += v[1]
						}
						cx /= float64(len(f.Points))
						cy /= float64(len(f.Points))
						s := .4 + .6*p
						angle := (1 - p) * .5
						if index%2 == 0 {
							angle = -angle
						}
						centerX := 35.5 + (cx-35.5)*(1+.30*(1-p))
						centerY := 24.5 + (cy-24.5)*(1+.30*(1-p))
						dx, dy := x-centerX, y-centerY
						sx = cx + (dx*math.Cos(angle)+dy*math.Sin(angle))/s
						sy = cy + (-dx*math.Sin(angle)+dy*math.Cos(angle))/s
					}
					if inside(f.Points, sx, sy) {
						bits |= 1 << sub
						co := faceColor(f, sx, sy)
						for k := range total {
							total[k] += co[k]
						}
						hits++
						break
					}
				}
			}
			if hits == 0 {
				continue
			}
			for k := range total {
				total[k] /= hits
			}
			ch := glyphs[bits]
			if t < 1.45 && bits != 255 && ((r*17+col*13+int(t*25))%7 == 0 || t < .6) {
				ch = []rune("01{}<>/+*")[(r+col+int(t*30))%9]
				total = mix(total, cyan, .65)
			}
			out[position{top + r, xoff + col}] = pixel{ch, total}
		}
	}
}
func (l layout) title(out canvas, top int, t float64) {
	if l.wordRows == 0 {
		out.text(top, l.center(14), "TOKENS INFINITY", ink)
		return
	}
	a := art.Words[strconv.Itoa(l.wordRows)]
	bits := make([]*big.Int, len(a.Bits))
	for i, s := range a.Bits {
		bits[i] = new(big.Int)
		bits[i].SetString(s, 16)
	}
	for r := 0; r < l.wordRows; r++ {
		for c := 0; c < l.wordCols; c++ {
			mask := 0
			for s := 0; s < 8; s++ {
				if bits[r*4+s/2].Bit(c*2+s%2) != 0 {
					mask |= 1 << s
				}
			}
			if mask == 0 {
				continue
			}
			lock := 1.15 + .62*float64(c)/float64(l.wordCols) + float64((c*31+r*11)%10)*.01
			if t < lock-.34 {
				continue
			}
			ch := glyphs[mask]
			color := ink
			if t < lock {
				ch = []rune("01{}<>/+*")[(r+c+int(t*30))%9]
				color = mix(bg, cyan, (t-lock+.34)/.2)
			} else {
				color = mix(rgb{255, 255, 255}, ink, (t-lock)/.22)
			}
			out[position{top + r, l.center(l.wordCols) + c}] = pixel{ch, color}
		}
	}
}
func (l layout) frame(top int, t float64) canvas {
	c := canvas{}
	if l.wordRows == 0 {
		l.title(c, top, t)
		return c
	}
	if t < .3 {
		if t < .11 || t >= .19 {
			c[position{top + l.logoRows/2, l.cols / 2}] = pixel{'█', rgb{0, 109, 255}}
		}
	} else {
		l.logo(c, top, t)
	}
	l.title(c, top+l.wordTop, t)
	sweep := (t - 2.02) / .7
	if sweep > 0 && sweep < 1 {
		center := -.25 + 1.5*sweep*sweep*(3-2*sweep)
		for p, v := range c {
			u := (float64(p.c-l.center(l.width))/float64(max(1, l.width)) + (1-float64(p.r-top)/float64(max(1, l.wordTop+l.wordRows)))*.45) / 1.45
			glow := 1 - math.Abs(u-center)/.085
			if glow > 0 {
				v.color = mix(v.color, rgb{211, 248, 255}, .7*glow*glow)
				c[p] = v
			}
		}
	}
	return c
}
func (l layout) home(top int) canvas {
	c := canvas{}
	l.logo(c, top, 3)
	l.title(c, top+l.wordTop, 3)
	if l.wordRows > 0 {
		c.text(top+l.sloganTop, l.center(16), "从词元，抵达无限", ink)
	}
	tag := "T O K E N S   I N F I N I T Y"
	c.text(top+l.tagTop, l.center(textWidth(tag)), tag, muted)
	line := "──  词元无限 · " + brandLabel + "  ──"
	c.text(top+l.platformTop, l.center(textWidth(line)), line, rgb{0, 145, 255})
	return c
}
func render(c canvas, cols, rows int) string {
	keys := make([]position, 0, len(c))
	for p := range c {
		if p.r >= 0 && p.r < rows && p.c >= 0 && p.c+charWidth(c[p].ch) <= cols {
			keys = append(keys, p)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].r != keys[j].r {
			return keys[i].r < keys[j].r
		}
		return keys[i].c < keys[j].c
	})
	var b strings.Builder
	last := position{-1, -1}
	lastColor := rgb{-1, -1, -1}
	for _, p := range keys {
		v := c[p]
		if last != p {
			fmt.Fprintf(&b, "\x1b[%d;%dH", p.r+1, p.c+1)
		}
		if v.color != lastColor {
			fmt.Fprintf(&b, "\x1b[22;38;2;%d;%d;%dm", v.color[0], v.color[1], v.color[2])
			lastColor = v.color
		}
		b.WriteRune(v.ch)
		last = position{p.r, p.c + charWidth(v.ch)}
	}
	b.WriteString("\x1b[0m")
	return b.String()
}
