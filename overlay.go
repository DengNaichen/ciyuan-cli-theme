package main

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Track only app cells. Paint known-empty rows; erase before app output touches
// them. Unsupported screen operations suspend painting until a full clear.
type overlay struct {
	app, typed                                                              string
	submitted                                                               bool
	cols, rows, cx, cy, sx, sy, helpRow                                     int
	grid                                                                    [][]bool
	count                                                                   []int
	visible                                                                 map[int]bool
	alt, entered, awaiting, drop, known, sync, dirty, disabled, bg, reverse bool
	carry                                                                   []byte
	anchor                                                                  *layout
	top                                                                     int
}

func newOverlay(cols, rows int, fromIntro bool) *overlay {
	o := &overlay{alt: fromIntro, awaiting: fromIntro, drop: fromIntro}
	o.resize(cols, rows)
	return o
}
func (o *overlay) resize(cols, rows int) {
	o.cols, o.rows = max(1, cols), max(1, rows)
	o.grid = make([][]bool, o.rows)
	o.count = make([]int, o.rows)
	for i := range o.grid {
		o.grid[i] = make([]bool, o.cols)
	}
	o.cx, o.cy = 0, 0
	o.known = false
	o.visible = map[int]bool{}
	o.anchor = nil
	o.helpRow = -1
}
func wrap(s string) string {
	if s == "" {
		return ""
	}
	return "\x1b7\x1b[0m" + s + "\x1b[0m\x1b8"
}
func (o *overlay) erase(lo, hi int) string {
	var b strings.Builder
	for r := max(0, lo); r < min(o.rows, hi); r++ {
		if o.visible[r] {
			fmt.Fprintf(&b, "\x1b[%d;1H\x1b[2K", r+1)
			delete(o.visible, r)
		}
	}
	return wrap(b.String())
}
func (o *overlay) uncertain() string       { o.known = false; o.dirty = true; return o.erase(0, o.rows) }
func (o *overlay) touch(lo, hi int) string { o.dirty = true; return o.erase(lo, hi) }
func (o *overlay) span(r, a, b int, filled bool) {
	if r < 0 || r >= o.rows {
		return
	}
	for c := max(0, a); c < min(o.cols, b); c++ {
		if o.grid[r][c] != filled {
			if filled {
				o.count[r]++
			} else {
				o.count[r]--
			}
			o.grid[r][c] = filled
		}
	}
}
func (o *overlay) clear(lo, hi int, filled bool) {
	for r := max(0, lo); r < min(o.rows, hi); r++ {
		o.span(r, 0, o.cols, filled)
	}
}
func (o *overlay) frameEnd() string {
	if o.disabled || !o.alt {
		return ""
	}
	if !o.known {
		return o.erase(0, o.rows)
	}
	used := []int{}
	for r, n := range o.count {
		if n > 0 {
			used = append(used, r)
		}
	}
	if len(used) < 2 || used[0] > 2 {
		return o.erase(0, o.rows)
	}
	limit := 4
	if o.app == "claude" && !o.submitted {
		limit = 10
	}
	top := used[0]
	for _, r := range used[1:] {
		if r <= limit && r-top <= 2 {
			top = r
		} else {
			break
		}
	}
	if o.helpRow > top && o.helpRow <= top+3 {
		for _, r := range used {
			if r >= o.helpRow && r <= o.helpRow+7 {
				top = r
			}
		}
	}
	bottom := used[len(used)-1]
	for i := len(used) - 2; i >= 0; i-- {
		r := used[i]
		if r > top && bottom-r <= 2 {
			bottom = r
		} else {
			break
		}
	}
	if bottom <= top {
		return o.erase(0, o.rows)
	}
	for _, r := range used {
		if r > top && r < bottom {
			return o.erase(0, o.rows)
		}
	}
	out := ""
	if o.anchor != nil && (top+2 > o.top || o.top+o.anchor.height > bottom-1) {
		out = o.erase(0, o.rows)
		o.anchor = nil
	}
	if o.anchor == nil {
		gap := max(5, top+2)
		l := choose(o.cols, bottom-1-gap)
		if l.height > bottom-1-gap {
			return out + o.erase(0, o.rows)
		}
		o.anchor = &l
		o.top = gap + max(0, (bottom-1-gap-l.height)/2)
	}
	content := o.anchor.home(o.top)
	want := map[int]bool{}
	for p := range content {
		if p.r > top && p.r < bottom-1 && p.r >= 0 && p.r < o.rows && o.count[p.r] == 0 {
			want[p.r] = true
		}
	}
	for r := range o.visible {
		if !want[r] {
			out += o.erase(r, r+1)
		}
	}
	draw := canvas{}
	missing := []int{}
	for r := range want {
		if !o.visible[r] {
			missing = append(missing, r)
			for p, v := range content {
				if p.r == r {
					draw[p] = v
				}
			}
			o.visible[r] = true
		}
	}
	if len(missing) > 0 {
		sort.Ints(missing)
		var b strings.Builder
		for _, r := range missing {
			fmt.Fprintf(&b, "\x1b[%d;1H\x1b[2K", r+1)
		}
		b.WriteString(render(draw, o.cols, o.rows))
		out += wrap(b.String())
	}
	return out
}
func (o *overlay) leaveIntro() string {
	o.awaiting = false
	o.drop = false
	o.alt = false
	o.known = false
	o.visible = map[int]bool{}
	return "\x1b[?1049l"
}
func (o *overlay) text(s string) string {
	if o.awaiting {
		return o.leaveIntro()
	}
	if strings.Contains(s, "To get started") {
		o.helpRow = o.cy
	}
	var out strings.Builder
	for _, r := range s {
		w := charWidth(r)
		if w == 0 {
			continue
		}
		if o.cx+w > o.cols {
			o.cx = 0
			o.cy++
		}
		if o.cy >= o.rows {
			o.cy = o.rows - 1
			out.WriteString(o.uncertain())
		}
		out.WriteString(o.touch(o.cy, o.cy+1))
		o.span(o.cy, o.cx, o.cx+w, o.bg || o.reverse || r != ' ')
		o.cx += w
	}
	return out.String()
}
func (o *overlay) control(c byte) string {
	if o.awaiting && strings.ContainsRune("\n\v\f\r\b\t", rune(c)) {
		return o.leaveIntro()
	}
	switch c {
	case '\n', '\v', '\f':
		if o.cy+1 >= o.rows {
			return o.uncertain()
		}
		o.cy++
	case '\r':
		o.cx = 0
	case '\b':
		o.cx = max(0, o.cx-1)
	case '\t':
		o.cx = min(o.cols-1, (o.cx/8+1)*8)
	}
	return ""
}
func (o *overlay) save() string {
	if o.alt && o.entered {
		o.disabled = true
		return o.erase(0, o.rows)
	}
	o.sx, o.sy = o.cx, o.cy
	return ""
}
func (o *overlay) sgr(body string) {
	ps := strings.Split(body, ";")
	for i := 0; i < len(ps); i++ {
		p := ps[i]
		v, _ := strconv.Atoi(strings.Split(p, ":")[0])
		switch {
		case v == 0:
			o.bg = false
			o.reverse = false
		case v == 7:
			o.reverse = true
		case v == 27:
			o.reverse = false
		case v >= 40 && v <= 47 || v >= 100 && v <= 107:
			o.bg = true
		case v == 49:
			o.bg = false
		}
		if v == 38 || v == 48 || v == 58 {
			if v == 48 {
				o.bg = true
			}
			if !strings.Contains(p, ":") && i+1 < len(ps) {
				if ps[i+1] == "5" {
					i += 2
				} else if ps[i+1] == "2" {
					i += 4
				}
			}
		}
	}
}
func (o *overlay) escape(s string) (pre, replacement string) {
	replacement = s
	if len(s) < 2 {
		return
	}
	switch s[1] {
	case '[':
		return o.csi(s)
	case '7':
		pre = o.save()
	case '8':
		o.cx, o.cy = o.sx, o.sy
	case 'M':
		if o.cy == 0 {
			pre = o.uncertain()
		} else {
			o.cy--
		}
	case 'D', 'E':
		if o.cy+1 >= o.rows {
			pre = o.uncertain()
		} else {
			o.cy++
		}
		if s[1] == 'E' {
			o.cx = 0
		}
	case 'c':
		pre = o.uncertain()
		o.alt = false
	}
	return
}
func (o *overlay) csi(s string) (pre, replacement string) {
	replacement = s
	body := s[2 : len(s)-1]
	final := s[len(s)-1]
	private := byte(0)
	if len(body) > 0 && strings.ContainsRune("?<>=", rune(body[0])) {
		private = body[0]
		body = body[1:]
	}
	for _, r := range body {
		if r >= 0x20 && r <= 0x2f {
			return
		}
	}
	if final == 'm' && private == 0 {
		o.sgr(body)
		return
	}
	ns := []int{}
	if body != "" {
		for _, p := range strings.Split(body, ";") {
			part := strings.Split(p, ":")[0]
			if part == "" {
				ns = append(ns, 0)
				continue
			}
			n, e := strconv.Atoi(part)
			if e != nil {
				if private == 0 {
					pre = o.uncertain()
				}
				return
			}
			ns = append(ns, n)
		}
	}
	a, b := 0, 0
	if len(ns) > 0 {
		a = ns[0]
	}
	if len(ns) > 1 {
		b = ns[1]
	}
	if private == '?' {
		if final != 'h' && final != 'l' {
			return
		}
		on := final == 'h'
		for _, m := range ns {
			switch m {
			case 1049, 1047, 47:
				if on {
					o.awaiting = false
					if o.drop && len(ns) == 1 {
						o.drop = false
						replacement = "\x1b[2J"
						o.clear(0, o.rows, false)
						o.known = true
					} else {
						o.known = false
					}
					o.alt = true
					o.entered = true
					o.visible = map[int]bool{}
				} else {
					o.alt = false
					o.known = false
					o.visible = map[int]bool{}
				}
				o.dirty = true
			case 2026:
				o.sync = on
				if !on && o.dirty {
					o.dirty = false
					pre += o.frameEnd()
				}
			}
		}
		return
	}
	if private != 0 {
		return
	}
	if o.awaiting && strings.ContainsRune("HfABCDEFG`dJKXL@PMST", rune(final)) {
		pre = o.leaveIntro()
		return
	}
	n := max(1, a)
	switch final {
	case 'H', 'f':
		o.cy = min(o.rows-1, max(0, a-1))
		o.cx = min(o.cols-1, max(0, b-1))
	case 'A':
		o.cy = max(0, o.cy-n)
	case 'B', 'e':
		o.cy = min(o.rows-1, o.cy+n)
	case 'C', 'a':
		o.cx = min(o.cols-1, o.cx+n)
	case 'D':
		o.cx = max(0, o.cx-n)
	case 'E':
		o.cy = min(o.rows-1, o.cy+n)
		o.cx = 0
	case 'F':
		o.cy = max(0, o.cy-n)
		o.cx = 0
	case 'G', '`':
		o.cx = min(o.cols-1, n-1)
	case 'd':
		o.cy = min(o.rows-1, n-1)
	case 'J':
		fill := o.bg || o.reverse
		switch a {
		case 0:
			pre = o.touch(o.cy, o.rows)
			o.span(o.cy, o.cx, o.cols, fill)
			o.clear(o.cy+1, o.rows, fill)
			if o.cx == 0 && o.cy == 0 && o.alt {
				o.known = true
				o.helpRow = -1
				o.submitted = false
			}
		case 1:
			pre = o.touch(0, o.cy+1)
			o.clear(0, o.cy, fill)
			o.span(o.cy, 0, o.cx+1, fill)
		case 2:
			pre = o.touch(0, o.rows)
			o.clear(0, o.rows, fill)
			if o.alt {
				o.known = true
				o.helpRow = -1
				o.submitted = false
			}
		}
	case 'K':
		pre = o.touch(o.cy, o.cy+1)
		switch a {
		case 0:
			o.span(o.cy, o.cx, o.cols, o.bg || o.reverse)
		case 1:
			o.span(o.cy, 0, o.cx+1, o.bg || o.reverse)
		default:
			o.span(o.cy, 0, o.cols, o.bg || o.reverse)
		}
	case 'X':
		pre = o.touch(o.cy, o.cy+1)
		o.span(o.cy, o.cx, o.cx+n, o.bg || o.reverse)
	case 'r':
		o.cx, o.cy = 0, 0
		if max(1, a) != 1 || (b != 0 && b != o.rows) {
			pre = o.uncertain()
		}
	case 's':
		if body == "" {
			pre = o.save()
		}
	case 'u':
		if body == "" {
			o.cx, o.cy = o.sx, o.sy
		}
	case '@', 'P', 'L', 'M', 'S', 'T', 'b':
		pre = o.uncertain()
	}
	return
}
func sequenceEnd(data []byte, i int) int {
	n := len(data)
	if i+1 >= n {
		return -1
	}
	kind := data[i+1]
	if kind == '[' {
		j := i + 2
		for j < n && data[j] >= 0x20 && data[j] <= 0x3f {
			j++
		}
		if j >= n {
			return -1
		}
		return j + 1
	}
	if strings.ContainsRune("]P_^X", rune(kind)) {
		for j := i + 2; j < n; j++ {
			if kind == ']' && data[j] == 7 {
				return j + 1
			}
			if data[j] == 27 && j+1 < n && data[j+1] == '\\' {
				return j + 2
			}
		}
		return -1
	}
	if strings.ContainsRune("()*+-./# %", rune(kind)) {
		if i+2 < n {
			return i + 3
		}
		return -1
	}
	return i + 2
}
func (o *overlay) feed(chunk []byte) string {
	data := append(o.carry, chunk...)
	o.carry = nil
	name := []byte("OpenAI Codex")
	if o.app == "claude" {
		name = []byte("Claude Code")
	}
	var held []byte
	for k := min(len(name)-1, len(data)); k > 0; k-- {
		if bytes.Equal(data[len(data)-k:], name[:k]) {
			held = append([]byte(nil), data[len(data)-k:]...)
			data = data[:len(data)-k]
			break
		}
	}
	data = bytes.ReplaceAll(data, name, []byte("词元无限   "))
	var out strings.Builder
	for i := 0; i < len(data); {
		if data[i] == 27 {
			j := sequenceEnd(data, i)
			if j < 0 {
				o.carry = append([]byte(nil), data[i:]...)
				break
			}
			pre, repl := o.escape(string(data[i:j]))
			out.WriteString(pre)
			out.WriteString(repl)
			i = j
		} else if data[i] < 32 || data[i] == 127 {
			out.WriteString(o.control(data[i]))
			out.WriteByte(data[i])
			i++
		} else {
			j := i
			for j < len(data) && data[j] >= 32 && data[j] != 127 {
				_, size := utf8.DecodeRune(data[j:])
				if size == 1 && data[j] >= 128 && !utf8.FullRune(data[j:]) {
					break
				}
				j += size
			}
			if j == i {
				o.carry = append([]byte(nil), data[i:]...)
				break
			}
			s := string(data[i:j])
			out.WriteString(o.text(s))
			out.WriteString(s)
			i = j
		}
	}
	o.carry = append(o.carry, held...)
	if len(o.carry) > 1<<20 {
		out.WriteString(o.uncertain())
		o.disabled = true
		out.Write(o.carry)
		o.carry = nil
	}
	if !o.sync && o.dirty {
		o.dirty = false
		if extra := o.frameEnd(); extra != "" {
			out.WriteString("\x1b[?2026h" + extra + "\x1b[?2026l")
		}
	}
	return out.String()
}

func (o *overlay) noteInput(data []byte) {
	if o.app != "claude" {
		return
	}
	for i := 0; i < len(data); {
		if data[i] == 27 {
			j := sequenceEnd(data, i)
			if j < 0 {
				return
			}
			i = j
			continue
		}
		r, n := utf8.DecodeRune(data[i:])
		i += n
		switch r {
		case '\r':
			o.submitted = true
			if strings.TrimSpace(o.typed) == "/clear" || strings.TrimSpace(o.typed) == "/new" {
				o.submitted = false
			}
			o.typed = ""
		case 127, 8:
			rs := []rune(o.typed)
			if len(rs) > 0 {
				o.typed = string(rs[:len(rs)-1])
			}
		case 21, 3:
			o.typed = ""
		default:
			if r >= 32 {
				o.typed += string(r)
				if len(o.typed) > 128 {
					o.typed = o.typed[len(o.typed)-128:]
				}
			}
		}
	}
}
