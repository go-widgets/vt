package vt

import "testing"

func fed(cols, rows int, in string) *Term {
	t := New(cols, rows)
	t.WriteString(in)
	return t
}

func TestColorAccessors(t *testing.T) {
	if !Color(0).IsDefault() || Indexed(5).IsDefault() || RGB(0, 0, 1).IsDefault() {
		t.Fatal("IsDefault is wrong")
	}
	if i, ok := Indexed(5).Index(); !ok || i != 5 {
		t.Fatalf("Index of Indexed(5) = %d, %v", i, ok)
	}
	if _, ok := RGB(1, 2, 3).Index(); ok {
		t.Fatal("a direct colour has no index")
	}
	if r, g, b, ok := RGB(1, 2, 3).Direct(); !ok || r != 1 || g != 2 || b != 3 {
		t.Fatalf("Direct of RGB(1,2,3) = %d %d %d %v", r, g, b, ok)
	}
	if _, _, _, ok := Indexed(5).Direct(); ok {
		t.Fatal("an indexed colour is not direct")
	}
}

func TestCellOutsideAndInside(t *testing.T) {
	tm := fed(4, 2, "x")
	if c := tm.Cell(0, 0); c.R != 'x' {
		t.Fatalf("Cell(0,0) = %q", c.R)
	}
	for _, p := range [][2]int{{-1, 0}, {0, -1}, {2, 0}, {0, 4}} {
		if c := tm.Cell(p[0], p[1]); c != (Cell{R: ' '}) {
			t.Fatalf("Cell%v outside the screen = %+v", p, c)
		}
	}
}

func TestResizeSameAndOnAlternateScreen(t *testing.T) {
	tm := fed(4, 2, "ab")
	if !tm.Resize(4, 2) {
		t.Fatal("a resize to the same size is refused")
	}
	tm.WriteString("\x1b[?1049h")
	if !tm.Resize(6, 3) {
		t.Fatal("resize refused")
	}
	tm.WriteString("z")
	if got := screen(tm); got != "  z\n\n\n" {
		t.Fatalf("alternate screen after a resize: %q", got)
	}
	tm.WriteString("\x1b[?1049l")
	if got := screen(tm); got != "ab\n\n\n" {
		t.Fatalf("main screen after a resize: %q", got)
	}
}

func TestSequencesAndStates(t *testing.T) {
	cases := []struct {
		name, in, want string
		cols, rows     int
	}{
		{"second intermediate after ESC", "\x1b((0q", "─\n", 5, 1},
		{"CSI intermediates, final ignored", "\x1b[ !pX", "X\n", 5, 1},
		{"CSI intermediate then a bad byte", "\x1b[ 0abc", "bc\n", 5, 1},
		{"CSI ignore swallows non-final bytes", "\x1b[ 01xyz", "yz\n", 5, 1},
		{"private marker after a number", "\x1b[1?mZ", "Z\n", 5, 1},
		{"backspace", "ab\bx", "ax\n", 5, 1},
		{"backspace at column 0", "\bx", "x\n", 5, 1},
		{"ESC inside a string", "abc\x1b]t\x1b\x1b[1;2HZ", "aZc\n", 5, 1},
		{"ESC D", "a\x1bDb", "a\n b\n", 5, 2},
		{"ESC M scrolls down at the top", "ab\r\ncd\x1b[1;1H\x1bM", "\nab\ncd\n", 5, 3},
		{"ESC M moves up", "ab\r\ncd\x1bMx", "abx\ncd\n", 5, 2},
		{"ED 1", "abc\r\ndef\x1b[2;2H\x1b[1J", "\n  f\n", 5, 2},
		{"IL outside the margins", "a\r\nb\r\nc\x1b[2;3r\x1b[L", "a\nb\nc\n\n", 5, 4},
		{"DL outside the margins", "a\r\nb\r\nc\x1b[2;3r\x1b[M", "a\nb\nc\n\n", 5, 4},
		{"combining mark is dropped", "áb", "ab\n", 5, 1},
		{"wide char, no autowrap, narrow screen", "\x1b[?7la日", "日\n", 2, 1},
		{"wide char that does not fit wraps", "ab日", "ab\n日\n", 3, 2},
		{"wide char on a one-column screen", "日", "\n\n", 1, 2},
		{"wide char on a one-column screen, no autowrap", "\x1b[?7l日", "\n", 1, 1},
		{"overwriting a tail blanks the head", "日\x1b[1;2Hx", " x\n", 4, 1},
		{"overwriting a head blanks the tail", "日\x1b[1;1Hx", "x\n", 4, 1},
		{"a wide char over the head of another blanks its tail", "\x1b[1;2H日\x1b[1;1H日", "日\n", 6, 1},
		{"alternate screen keeps its content with 47", "\x1b[?47hzz\x1b[?47l\x1b[?47h", "zz\n", 5, 1},
		{"1047 clears the alternate screen on leaving", "\x1b[?1047hzz\x1b[?1047l\x1b[?47h", "\n", 5, 1},
		{"leaving the alternate screen twice is a no-op", "ab\x1b[?1049l", "ab\n", 5, 1},
	}
	for _, c := range cases {
		if got := screen(fed(c.cols, c.rows, c.in)); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestCursorMoves(t *testing.T) {
	cases := []struct {
		name, in string
		x, y     int
	}{
		{"CUU", "\x1b[3;4H\x1b[A", 3, 1},
		{"CUD", "\x1b[3;4H\x1b[2B", 3, 4},
		{"VPR", "\x1b[e", 0, 1},
		{"CUB", "\x1b[3;4H\x1b[2D", 1, 2},
		{"HPR", "\x1b[3;4H\x1b[a", 4, 2},
		{"CNL", "\x1b[3;4H\x1b[E", 0, 3},
		{"CPL", "\x1b[3;4H\x1b[F", 0, 1},
		{"CHA", "\x1b[5G", 4, 0},
		{"HPA", "\x1b[3`", 2, 0},
		{"CHT", "\x1b[I", 8, 0},
		{"CBT", "\x1b[1;12H\x1b[Z", 8, 0},
		{"CBT twice", "\x1b[1;12H\x1b[2Z", 0, 0},
		{"VPA", "\x1b[3d", 0, 2},
		{"VPA in origin mode", "\x1b[2;4r\x1b[?6h\x1b[2d", 0, 2},
		{"TBC 0", "\x1b[1;9H\x1b[g\x1b[1;1H\x1b[I", 16, 0},
		{"TBC 3", "\x1b[3g\x1b[I", 19, 0},
		{"HTS", "\x1b[1;4H\x1bH\x1b[1;1H\x1b[I", 3, 0},
		{"SCOSC SCORC", "\x1b[3;4H\x1b[s\x1b[1;1H\x1b[u", 3, 2},
		{"DECSC in the alternate screen", "\x1b[?47h\x1b[2;2H\x1b7\x1b[1;1H\x1b8", 1, 1},
		{"CUD outside the margins is not held by them", "\x1b[3;4r\x1b[B", 0, 1},
		{"CUD inside the margins stops at the bottom", "\x1b[3;4r\x1b[3;1H\x1b[5B", 0, 3},
	}
	for _, c := range cases {
		tm := fed(20, 5, c.in)
		if x, y, _ := tm.Cursor(); x != c.x || y != c.y {
			t.Errorf("%s: cursor at %d,%d, want %d,%d", c.name, x, y, c.x, c.y)
		}
	}
}

func TestCursorVisibility(t *testing.T) {
	tm := fed(5, 1, "\x1b[?25l")
	if _, _, vis := tm.Cursor(); vis {
		t.Fatal("the cursor is still shown")
	}
	tm.WriteString("\x1b[?25h")
	if _, _, vis := tm.Cursor(); !vis {
		t.Fatal("the cursor is hidden")
	}
}

func TestSGRAttributes(t *testing.T) {
	cases := []struct {
		in   string
		want Attr
	}{
		{"\x1b[2m", Faint},
		{"\x1b[8m", Hidden},
		{"\x1b[21m", Underline},
		{"\x1b[3;23m", 0},
		{"\x1b[4;24m", 0},
		{"\x1b[7;27m", 0},
		{"\x1b[8;28m", 0},
		{"\x1b[38;9;1m", 0},
	}
	for _, c := range cases {
		if got := fed(3, 1, c.in+"x").Cell(0, 0); got.A != c.want || got.FG != 0 {
			t.Errorf("%q: attributes %b, fg %v; want %b and the default colour", c.in, got.A, got.FG, c.want)
		}
	}
	if got := fed(3, 1, "\x1b[41;49mx").Cell(0, 0); got.BG != 0 {
		t.Errorf("49 left the background %v", got.BG)
	}
}

func TestDefensiveGuards(t *testing.T) {
	tm := New(4, 2)
	tm.TakeDirty()
	w := tm.Work()
	tm.fill(0, 3, 3)
	if d := tm.TakeDirty(); len(d) != 0 || tm.Work() != w {
		t.Fatalf("an empty fill touched the screen: dirty %v, work %d -> %d", d, w, tm.Work())
	}
	tm.charsets = [2]byte{'B', 'B'}
	tm.inter = '('
	tm.escInter(0x20)
	if tm.charsets != [2]byte{'B', 'B'} || tm.st != stGround {
		t.Fatalf("an intermediate byte designated a set: %v", tm.charsets)
	}
	if c, ok := extColor(nil); ok || c != 0 {
		t.Fatalf("extColor(nil) = %v, %v", c, ok)
	}
	tm.g[0][0] = Cell{Wide: 1}
	if r := tm.Runs(0); len(r) != 1 || r[0].Text != " " {
		t.Fatalf("a cell without a character is drawn as %+v", r)
	}
}

func TestWidthAndRunsBounds(t *testing.T) {
	for _, r := range []rune{0x07, 0x1f, 0x7f, 0x301, 0x200b} {
		if w := Width(r); w != 0 {
			t.Errorf("Width(%U) = %d, want 0", r, w)
		}
	}
	for r, want := range map[rune]int{'a': 1, 'é': 1, '日': 2, '😀': 2, 0x2603: 1} {
		if w := Width(r); w != want {
			t.Errorf("Width(%U) = %d, want %d", r, w, want)
		}
	}
	tm := New(3, 2)
	if tm.Runs(-1) != nil || tm.Runs(2) != nil {
		t.Fatal("Runs outside the screen is not nil")
	}
}
