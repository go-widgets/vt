# vt

A terminal emulator as a pure model: the bytes a program wrote to a terminal
in, the screen they leave out, as cells with a character, a colour and
attributes. It imports only the standard library and writes nothing back, so a
hostile stream can change the screen and nothing else.

```sh
go get github.com/go-widgets/vt
```

## What it does

- **Parser:** the state machine of ECMA-48 clause 5.4, with the xterm subset a
  screen needs.
  - C0 controls BS HT LF VT FF CR SO SI.
  - ESC 7 8 c D E M H, and the DEC special graphics set.
  - CSI cursor movement, erase, insert/delete, scroll, tab, repeat and margins.
  - Private modes 6 7 25 47 1047 1049, and insert mode 4 (alternate screen
    included).
  - SGR with 16, 256 and direct colours, in the semicolon and colon forms.
- **Answers nothing:** no answerbacks, no title, no clipboard. OSC, DCS, APC,
  PM and SOS strings are read to their terminator and dropped. The 8-bit C1
  controls are not recognised, so UTF-8 text cannot start a sequence.
- **Bounded:** every operation is limited by the screen's size, and `Work`
  counts the cells a stream made it touch, so a caller can refuse one that does
  far more than it was given bytes.
- **Wide characters:** `Width` counts columns; CJK and emoji take two cells.
  Combining marks are dropped (a cell holds one code point).
- **Cheap to draw:** `TakeDirty` lists the rows changed since the last call, and
  `Runs` turns a row into runs of equal style.

```go
t := vt.New(80, 24)
t.WriteString("\x1b[1;31mred\x1b[0m\r\n")
for y := 0; y < t.Rows(); y++ {
    for _, r := range t.Runs(y) {
        draw(r.Text, r.FG, r.BG, r.A)
    }
}
```

## Origin

It moved here from `internal/vt` of `github.com/go-slides/slides`, with its
history (`git subtree split`), where it renders asciicast recordings. It is a
different emulator from `toolkit/vt` in go-widgets/toolkit (a scrollback and a
selection, no wide characters or alternate screen); folding the two into one is
tracked in go-widgets/toolkit.

## Tests

- 100 % statement coverage, gated in CI (it was 83.4 % in go-slides/slides).
- `FuzzWrite` fuzzes the parser for 45 s on every change; the corpus is
  committed in `testdata/fuzz/FuzzWrite`. It asserts the cursor stays on the
  screen and every run is valid UTF-8.
- `TestBombStaysBounded` asserts a scroll bomb is refused within its bounds.
- CI runs on Linux (amd64 and arm64), macOS, Windows, 32-bit Linux and
  `GOOS=js GOARCH=wasm`, cross-vets the six 64-bit Linux architectures, and
  runs govulncheck.

## Licence

BSD 3-Clause, see [LICENSE](LICENSE).
