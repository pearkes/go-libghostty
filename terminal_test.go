package libghostty

import (
	"fmt"
	"io"
	"testing"
)

func TestNewTerminalClose(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	term.Close()
}

func TestNewTerminalZeroDimensions(t *testing.T) {
	_, err := NewTerminal(WithSize(0, 24))
	if err == nil {
		t.Fatal("expected error for zero cols")
	}

	_, err = NewTerminal(WithSize(80, 0))
	if err == nil {
		t.Fatal("expected error for zero rows")
	}
}

func TestNewTerminalNoOptions(t *testing.T) {
	_, err := NewTerminal()
	if err == nil {
		t.Fatal("expected error when no size is specified")
	}
}

func TestNewTerminalWithScrollback(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24), WithMaxScrollbackLines(1000))
	if err != nil {
		t.Fatal(err)
	}
	term.Close()
}

func TestTerminalReset(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	// Write some data then reset; should not panic or error.
	term.VTWrite([]byte("hello"))
	term.Reset()
}

func TestTerminalResize(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	if err := term.Resize(120, 40, 8, 16); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalResizeZero(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	if err := term.Resize(0, 24, 8, 16); err == nil {
		t.Fatal("expected error for zero cols")
	}
}

func TestTerminalVTWrite(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	// Write plain text and escape sequences; should not panic.
	term.VTWrite([]byte("hello world"))
	term.VTWrite([]byte("\x1b[2J")) // clear screen
	term.VTWrite(nil)               // empty write
}

func TestTerminalIOWriter(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	// Terminal satisfies io.Writer.
	var w io.Writer = term
	n, err := fmt.Fprintf(w, "hello %s", "world")
	if err != nil {
		t.Fatal(err)
	}
	if n != len("hello world") {
		t.Fatalf("expected %d bytes written, got %d", len("hello world"), n)
	}
}

func TestTerminalModeGetSet(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	// Test several modes: set via ModeSet, read back via ModeGet.
	tests := []struct {
		name       string
		mode       Mode
		defaultVal bool
	}{
		{"CursorVisible", ModeCursorVisible, true},
		{"Wraparound", ModeWraparound, true},
		{"BackarrowKeyMode", ModeBackarrowKeyMode, false},
		{"BracketedPaste", ModeBracketedPaste, false},
		{"FocusEvent", ModeFocusEvent, false},
		{"AltScreen", ModeAltScreen, false},
		{"Origin", ModeOrigin, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify default value.
			val, err := term.ModeGet(tt.mode)
			if err != nil {
				t.Fatal(err)
			}
			if val != tt.defaultVal {
				t.Fatalf("expected default %v, got %v", tt.defaultVal, val)
			}

			// Toggle the mode.
			if err := term.ModeSet(tt.mode, !tt.defaultVal); err != nil {
				t.Fatal(err)
			}
			val, err = term.ModeGet(tt.mode)
			if err != nil {
				t.Fatal(err)
			}
			if val != !tt.defaultVal {
				t.Fatalf("expected %v after set, got %v", !tt.defaultVal, val)
			}

			// Toggle back.
			if err := term.ModeSet(tt.mode, tt.defaultVal); err != nil {
				t.Fatal(err)
			}
			val, err = term.ModeGet(tt.mode)
			if err != nil {
				t.Fatal(err)
			}
			if val != tt.defaultVal {
				t.Fatalf("expected %v after restore, got %v", tt.defaultVal, val)
			}
		})
	}
}

func TestTerminalModeVTWrite(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	// Test that VT escape sequences correctly set and reset modes
	// and that ModeGet reads them back.
	// DEC private modes use CSI ? <n> h (set) / CSI ? <n> l (reset).
	// ANSI modes use CSI <n> h (set) / CSI <n> l (reset).
	tests := []struct {
		name       string
		mode       Mode
		setSeq     string
		resetSeq   string
		defaultVal bool
	}{
		{"BracketedPaste", ModeBracketedPaste, "\x1b[?2004h", "\x1b[?2004l", false},
		{"BackarrowKeyMode", ModeBackarrowKeyMode, "\x1b[?67h", "\x1b[?67l", false},
		{"CursorVisible", ModeCursorVisible, "\x1b[?25h", "\x1b[?25l", true},
		{"FocusEvent", ModeFocusEvent, "\x1b[?1004h", "\x1b[?1004l", false},
		{"NormalMouse", ModeNormalMouse, "\x1b[?1000h", "\x1b[?1000l", false},
		{"SGRMouse", ModeSGRMouse, "\x1b[?1006h", "\x1b[?1006l", false},
		{"Insert", ModeInsert, "\x1b[4h", "\x1b[4l", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify default.
			val, err := term.ModeGet(tt.mode)
			if err != nil {
				t.Fatal(err)
			}
			if val != tt.defaultVal {
				t.Fatalf("expected default %v, got %v", tt.defaultVal, val)
			}

			// Set via VT escape sequence.
			term.VTWrite([]byte(tt.setSeq))
			val, err = term.ModeGet(tt.mode)
			if err != nil {
				t.Fatal(err)
			}
			if !val {
				t.Fatal("expected mode set after VT write set sequence")
			}

			// Reset via VT escape sequence.
			term.VTWrite([]byte(tt.resetSeq))
			val, err = term.ModeGet(tt.mode)
			if err != nil {
				t.Fatal(err)
			}
			if val {
				t.Fatal("expected mode reset after VT write reset sequence")
			}

			// Restore to default for next subtest.
			if tt.defaultVal {
				term.VTWrite([]byte(tt.setSeq))
			}
		})
	}
}

func TestTerminalModeChanged(t *testing.T) {
	type modeEvent struct {
		mode     Mode
		enabled  bool
		observed bool
	}

	var (
		term        *Terminal
		events      []modeEvent
		callbackErr error
		order       []byte
	)
	term, err := NewTerminal(
		WithSize(80, 24),
		WithModeChanged(func(got *Terminal, mode Mode, enabled bool) {
			observed, err := got.ModeGet(mode)
			if err != nil && callbackErr == nil {
				callbackErr = err
			}
			if got != term && callbackErr == nil {
				callbackErr = fmt.Errorf("callback terminal = %p, want %p", got, term)
			}
			events = append(events, modeEvent{mode, enabled, observed})
			if mode == ModeFocusEvent {
				if enabled {
					order = append(order, 'M')
				} else {
					order = append(order, 'm')
				}
			}
		}),
		WithBell(func(_ *Terminal) {
			order = append(order, 'B')
		}),
		WithTitleChanged(func(_ *Terminal) {
			order = append(order, 'T')
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	// DECSET reports a transition after the new value is observable.
	term.VTWrite([]byte("\x1b[?1004h"))
	if callbackErr != nil {
		t.Fatal(callbackErr)
	}
	if len(events) != 1 || events[0] != (modeEvent{ModeFocusEvent, true, true}) {
		t.Fatalf("DECSET events = %#v", events)
	}

	// A redundant DECSET is silent, while DECRST reports the reset value.
	term.VTWrite([]byte("\x1b[?1004h"))
	if len(events) != 1 {
		t.Fatalf("redundant DECSET produced events: %#v", events)
	}
	term.VTWrite([]byte("\x1b[?1004l"))
	if len(events) != 2 || events[1] != (modeEvent{ModeFocusEvent, false, false}) {
		t.Fatalf("DECRST events = %#v", events)
	}

	// DECSAVE is silent. DECRESTORE fires only when the restored value differs.
	term.VTWrite([]byte("\x1b[?1004s"))
	term.VTWrite([]byte("\x1b[?1004h"))
	term.VTWrite([]byte("\x1b[?1004r"))
	term.VTWrite([]byte("\x1b[?1004r"))
	if len(events) != 4 || events[2] != (modeEvent{ModeFocusEvent, true, true}) ||
		events[3] != (modeEvent{ModeFocusEvent, false, false}) {
		t.Fatalf("save/restore events = %#v", events)
	}

	// ANSI modes preserve their packed ANSI bit through the callback.
	term.VTWrite([]byte("\x1b[4h"))
	if len(events) != 5 || events[4] != (modeEvent{ModeInsert, true, true}) {
		t.Fatalf("ANSI mode events = %#v", events)
	}

	// Host-initiated mode changes and resets do not synthesize effects.
	if err := term.ModeSet(ModeFocusEvent, true); err != nil {
		t.Fatal(err)
	}
	if err := term.ModeSet(ModeFocusEvent, false); err != nil {
		t.Fatal(err)
	}
	term.Reset()
	if len(events) != 5 {
		t.Fatalf("host operations produced events: %#v", events)
	}

	// Effects from a single write stay in parser order.
	order = nil
	term.VTWrite([]byte("\x1b[?1004h\x07\x1b]2;mode order\x1b\\\x1b[?1004l"))
	if string(order) != "MBTm" {
		t.Fatalf("effect order = %q, want %q", order, "MBTm")
	}
}
