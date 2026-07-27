package libghostty

import (
	"bytes"
	"testing"
)

func TestTerminalSetTitle(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	if err := term.SetTitle("my terminal"); err != nil {
		t.Fatal(err)
	}

	// Clear the title.
	if err := term.SetTitle(""); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalSetPwd(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	if err := term.SetPwd("/tmp"); err != nil {
		t.Fatal(err)
	}

	if err := term.SetPwd(""); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalSetColors(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	white := &ColorRGB{R: 255, G: 255, B: 255}
	black := &ColorRGB{R: 0, G: 0, B: 0}

	if err := term.SetColorForeground(white); err != nil {
		t.Fatal(err)
	}
	if err := term.SetColorBackground(black); err != nil {
		t.Fatal(err)
	}
	if err := term.SetColorCursor(white); err != nil {
		t.Fatal(err)
	}

	// Clear colors.
	if err := term.SetColorForeground(nil); err != nil {
		t.Fatal(err)
	}
	if err := term.SetColorBackground(nil); err != nil {
		t.Fatal(err)
	}
	if err := term.SetColorCursor(nil); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalSetAPCMaxBytes(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	limit := uint(1024)
	if err := term.SetAPCMaxBytes(&limit); err != nil {
		t.Fatal(err)
	}
	if err := term.SetAPCMaxBytes(nil); err != nil {
		t.Fatal(err)
	}

	kittyLimit := uint(512)
	if err := term.SetAPCMaxBytesKitty(&kittyLimit); err != nil {
		t.Fatal(err)
	}
	if err := term.SetAPCMaxBytesKitty(nil); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalWithBell(t *testing.T) {
	var bellCount int
	term, err := NewTerminal(WithSize(80, 24), WithBell(func(_ *Terminal) {
		bellCount++
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	// BEL character should trigger the callback.
	term.VTWrite([]byte("\x07"))
	if bellCount != 1 {
		t.Fatalf("expected 1 bell, got %d", bellCount)
	}

	// Multiple BELs.
	term.VTWrite([]byte("\x07\x07"))
	if bellCount != 3 {
		t.Fatalf("expected 3 bells, got %d", bellCount)
	}
}

func TestTerminalSetEffectBell(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	var bellCount int
	term.SetEffectBell(func(_ *Terminal) {
		bellCount++
	})

	term.VTWrite([]byte("\x07"))
	if bellCount != 1 {
		t.Fatalf("expected 1 bell, got %d", bellCount)
	}

	// Clear the callback; bell should no longer fire.
	term.SetEffectBell(nil)
	term.VTWrite([]byte("\x07"))
	if bellCount != 1 {
		t.Fatalf("expected still 1 bell after clearing, got %d", bellCount)
	}
}

func TestTerminalWithClipboardWrite(t *testing.T) {
	var writes []ClipboardWrite
	term, err := NewTerminal(
		WithSize(80, 24),
		WithClipboardWrite(func(_ *Terminal, write ClipboardWrite) ClipboardWriteResult {
			writes = append(writes, write)
			return ClipboardWriteDenied
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	// OSC 52 is decoded before the effect runs, including binary data and
	// sequences split across multiple writes.
	term.VTWrite([]byte("\x1b]52;c;aGVs"))
	term.VTWrite([]byte("bG8Ad29ybGQ=\x1b\\"))
	if len(writes) != 1 {
		t.Fatalf("expected 1 clipboard write, got %d", len(writes))
	}
	if got := writes[0].Location; got != ClipboardLocationStandard {
		t.Fatalf("expected standard clipboard, got %d", got)
	}
	if got := len(writes[0].Contents); got != 1 {
		t.Fatalf("expected 1 clipboard representation, got %d", got)
	}
	content := writes[0].Contents[0]
	if got := content.MIME; got != "text/plain" {
		t.Fatalf("expected text/plain MIME type, got %q", got)
	}
	if want := []byte("hello\x00world"); !bytes.Equal(content.Data, want) {
		t.Fatalf("expected clipboard data %q, got %q", want, content.Data)
	}

	// An empty content list is a clear, while the normalized destination is
	// still preserved.
	term.VTWrite([]byte("\x1b]52;s;\x1b\\"))
	if len(writes) != 2 {
		t.Fatalf("expected 2 clipboard writes, got %d", len(writes))
	}
	if got := writes[1].Location; got != ClipboardLocationSelection {
		t.Fatalf("expected selection clipboard, got %d", got)
	}
	if got := len(writes[1].Contents); got != 0 {
		t.Fatalf("expected a clipboard clear, got %d representations", got)
	}

	// iTerm2 Copy uses the same protocol-neutral callback shape.
	term.VTWrite([]byte("\x1b]1337;Copy=:aVRlcm0=\x1b\\"))
	if len(writes) != 3 {
		t.Fatalf("expected 3 clipboard writes, got %d", len(writes))
	}
	if got := writes[2].Contents[0].Data; !bytes.Equal(got, []byte("iTerm")) {
		t.Fatalf("expected decoded iTerm2 data, got %q", got)
	}

	// Clipboard reads and malformed payloads are intentionally ignored.
	term.VTWrite([]byte("\x1b]52;c;?\x1b\\"))
	term.VTWrite([]byte("\x1b]52;c;%%%\x1b\\"))
	if len(writes) != 3 {
		t.Fatalf("expected ignored reads and malformed data, got %d writes", len(writes))
	}

	// Callback data is copied into Go memory and remains valid after later
	// terminal writes have reused libghostty's borrowed descriptors.
	if want := []byte("hello\x00world"); !bytes.Equal(writes[0].Contents[0].Data, want) {
		t.Fatalf("expected retained clipboard data %q, got %q", want, writes[0].Contents[0].Data)
	}
}

func TestTerminalSetEffectClipboardWrite(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	var writes int
	term.SetEffectClipboardWrite(func(_ *Terminal, write ClipboardWrite) ClipboardWriteResult {
		writes++
		if write.Location != ClipboardLocationPrimary {
			t.Errorf("expected primary clipboard, got %d", write.Location)
		}
		return ClipboardWriteSuccess
	})

	term.VTWrite([]byte("\x1b]52;p;eA==\x1b\\"))
	if writes != 1 {
		t.Fatalf("expected 1 clipboard write, got %d", writes)
	}

	// Clearing the callback takes effect immediately.
	term.SetEffectClipboardWrite(nil)
	term.VTWrite([]byte("\x1b]52;p;eA==\x1b\\"))
	if writes != 1 {
		t.Fatalf("expected still 1 clipboard write after clearing, got %d", writes)
	}
}

func TestTerminalWithDesktopNotification(t *testing.T) {
	var notifications []DesktopNotification
	var callbackTerminal *Terminal
	term, err := NewTerminal(
		WithSize(80, 24),
		WithDesktopNotification(func(got *Terminal, notification DesktopNotification) {
			callbackTerminal = got
			notifications = append(notifications, notification)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	// OSC 777 preserves separate title and body fields across PTY reads.
	term.VTWrite([]byte("\x1b]777;notify;Codex;Needs "))
	term.VTWrite([]byte("attention\x1b\\"))
	if len(notifications) != 1 {
		t.Fatalf("expected 1 desktop notification, got %d", len(notifications))
	}
	if callbackTerminal != term {
		t.Fatal("expected callback to receive the originating terminal")
	}
	if got := notifications[0]; got.Title != "Codex" || got.Body != "Needs attention" {
		t.Fatalf("unexpected OSC 777 notification: %#v", got)
	}

	// OSC 9 has no title and carries its payload as the body.
	term.VTWrite([]byte("\x1b]9;Build complete\x07"))
	if len(notifications) != 2 {
		t.Fatalf("expected 2 desktop notifications, got %d", len(notifications))
	}
	if got := notifications[1]; got.Title != "" || got.Body != "Build complete" {
		t.Fatalf("unexpected OSC 9 notification: %#v", got)
	}

	// Callback strings are Go-owned and survive parser-buffer reuse.
	if got := notifications[0]; got.Title != "Codex" || got.Body != "Needs attention" {
		t.Fatalf("expected retained notification, got %#v", got)
	}
}

func TestTerminalSetEffectDesktopNotification(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	var count int
	term.SetEffectDesktopNotification(func(_ *Terminal, notification DesktopNotification) {
		count++
		if notification.Body != "Ready" {
			t.Errorf("expected notification body Ready, got %q", notification.Body)
		}
	})

	term.VTWrite([]byte("\x1b]9;Ready\x1b\\"))
	if count != 1 {
		t.Fatalf("expected 1 desktop notification, got %d", count)
	}

	term.SetEffectDesktopNotification(nil)
	term.VTWrite([]byte("\x1b]9;Ignored\x1b\\"))
	if count != 1 {
		t.Fatalf("expected callback removal to take effect, got %d notifications", count)
	}
}

func TestTerminalWithProgressReport(t *testing.T) {
	var reports []ProgressReport
	term, err := NewTerminal(
		WithSize(80, 24),
		WithProgressReport(func(_ *Terminal, report ProgressReport) {
			reports = append(reports, report)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	cases := []struct {
		sequence string
		state    ProgressState
		progress int
	}{
		{"\x1b]9;4;0;\x1b\\", ProgressStateRemove, -1},
		{"\x1b]9;4;1;42\x07", ProgressStateSet, 42},
		{"\x1b]9;4;2;7\x1b\\", ProgressStateError, 7},
		{"\x1b]9;4;3\x1b\\", ProgressStateIndeterminate, -1},
		{"\x1b]9;4;4;75\x1b\\", ProgressStatePause, 75},
	}

	for i, test := range cases {
		midpoint := len(test.sequence) / 2
		term.VTWrite([]byte(test.sequence[:midpoint]))
		if len(reports) != i {
			t.Fatalf("report fired before sequence %d was complete", i)
		}
		term.VTWrite([]byte(test.sequence[midpoint:]))
		if len(reports) != i+1 {
			t.Fatalf("expected %d progress reports, got %d", i+1, len(reports))
		}

		got := reports[i]
		if got.State != test.state {
			t.Fatalf("report %d: expected state %d, got %d", i, test.state, got.State)
		}
		if test.progress < 0 {
			if got.Progress != nil {
				t.Fatalf("report %d: expected omitted progress, got %d", i, *got.Progress)
			}
		} else if got.Progress == nil || int(*got.Progress) != test.progress {
			t.Fatalf("report %d: expected progress %d, got %v", i, test.progress, got.Progress)
		}
	}

	// Optional progress values are Go-owned and survive later callbacks.
	if reports[1].Progress == nil || *reports[1].Progress != 42 {
		t.Fatalf("expected retained progress value 42, got %v", reports[1].Progress)
	}
}

func TestTerminalSetEffectProgressReport(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	var count int
	term.SetEffectProgressReport(func(_ *Terminal, report ProgressReport) {
		count++
		if report.State != ProgressStateSet {
			t.Errorf("expected set state, got %d", report.State)
		}
	})

	term.VTWrite([]byte("\x1b]9;4;1;90\x1b\\"))
	if count != 1 {
		t.Fatalf("expected 1 progress report, got %d", count)
	}

	term.SetEffectProgressReport(nil)
	term.VTWrite([]byte("\x1b]9;4;1;95\x1b\\"))
	if count != 1 {
		t.Fatalf("expected callback removal to take effect, got %d reports", count)
	}
}

func TestTerminalWithWritePty(t *testing.T) {
	var received []byte
	term, err := NewTerminal(WithSize(80, 24), WithWritePty(func(_ *Terminal, data []byte) {
		received = append(received, data...)
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	// DA1 query should produce a response via write_pty.
	term.VTWrite([]byte("\x1b[c"))
	if len(received) == 0 {
		t.Fatal("expected write_pty data from DA1 query")
	}
}

func TestTerminalWithTitleChanged(t *testing.T) {
	var titleChanged int
	term, err := NewTerminal(WithSize(80, 24), WithTitleChanged(func(_ *Terminal) {
		titleChanged++
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	// OSC 2 sets the title.
	term.VTWrite([]byte("\x1b]2;hello\x07"))
	if titleChanged != 1 {
		t.Fatalf("expected 1 title change, got %d", titleChanged)
	}
}

func TestTerminalWithEnquiry(t *testing.T) {
	var received []byte
	term, err := NewTerminal(
		WithSize(80, 24),
		WithWritePty(func(_ *Terminal, data []byte) {
			received = append(received, data...)
		}),
		WithEnquiry(func(_ *Terminal) []byte {
			return []byte("hello")
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	// ENQ character should trigger enquiry and write response via pty.
	term.VTWrite([]byte("\x05"))
	if string(received) != "hello" {
		t.Fatalf("expected enquiry response %q, got %q", "hello", string(received))
	}
}

func TestTerminalWithXtversion(t *testing.T) {
	var received []byte
	term, err := NewTerminal(
		WithSize(80, 24),
		WithWritePty(func(_ *Terminal, data []byte) {
			received = append(received, data...)
		}),
		WithXtversion(func(_ *Terminal) string {
			return "myterm 1.0"
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	// XTVERSION query: CSI > q
	term.VTWrite([]byte("\x1b[>q"))
	// Response should contain our version string in a DCS sequence.
	if len(received) == 0 {
		t.Fatal("expected xtversion response")
	}
	resp := string(received)
	if !contains(resp, "myterm 1.0") {
		t.Fatalf("expected response to contain %q, got %q", "myterm 1.0", resp)
	}
}

// contains reports whether s contains substr.
func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestTerminalSetColorPalette(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	// Set a custom palette.
	var palette Palette
	for i := range palette {
		palette[i] = ColorRGB{R: uint8(i), G: uint8(i), B: uint8(i)}
	}
	if err := term.SetColorPalette(&palette); err != nil {
		t.Fatal(err)
	}

	// Reset to default.
	if err := term.SetColorPalette(nil); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalSetScrollbackLimits(t *testing.T) {
	term, err := NewTerminal(
		WithSize(80, 24),
		WithMaxScrollbackBytes(4096),
		WithMaxScrollbackLines(100),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	bytes, err := term.ScrollbackMaxBytes()
	if err != nil {
		t.Fatal(err)
	}
	if bytes == nil || *bytes != 4096 {
		t.Fatalf("expected 4096-byte limit, got %v", bytes)
	}

	lines, err := term.ScrollbackMaxLines()
	if err != nil {
		t.Fatal(err)
	}
	if lines == nil || *lines != 100 {
		t.Fatalf("expected 100-line limit, got %v", lines)
	}

	if err := term.SetScrollbackMaxBytes(nil); err != nil {
		t.Fatal(err)
	}
	if err := term.SetScrollbackMaxLines(nil); err != nil {
		t.Fatal(err)
	}
	if bytes, err := term.ScrollbackMaxBytes(); err != nil {
		t.Fatal(err)
	} else if bytes != nil {
		t.Fatalf("expected unlimited byte limit, got %v", *bytes)
	}
	if lines, err := term.ScrollbackMaxLines(); err != nil {
		t.Fatal(err)
	} else if lines != nil {
		t.Fatalf("expected unlimited line limit, got %v", *lines)
	}
}

func TestTerminalAdditionalOptions(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	style := TerminalCursorStyleBar
	blink := true
	if err := term.SetDefaultCursorStyle(&style); err != nil {
		t.Fatal(err)
	}
	if err := term.SetDefaultCursorBlink(&blink); err != nil {
		t.Fatal(err)
	}
	if err := term.SetGlyphProtocol(true); err != nil {
		t.Fatal(err)
	}
	if err := term.SetDefaultCursorStyle(nil); err != nil {
		t.Fatal(err)
	}
	if err := term.SetDefaultCursorBlink(nil); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalPwdChangedEffect(t *testing.T) {
	var calls int
	var pwd string
	term, err := NewTerminal(
		WithSize(80, 24),
		WithPwdChanged(func(term *Terminal) {
			calls++
			pwd, _ = term.Pwd()
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	term.VTWrite([]byte("\x1b]7;file:///tmp\x07"))
	if calls != 1 {
		t.Fatalf("expected one pwd-changed callback, got %d", calls)
	}
	if pwd != "file:///tmp" {
		t.Fatalf("expected pwd %q, got %q", "file:///tmp", pwd)
	}

	term.SetEffectPwdChanged(nil)
	term.VTWrite([]byte("\x1b]7;file:///var\x07"))
	if calls != 1 {
		t.Fatalf("expected callback to remain cleared, got %d calls", calls)
	}
}

func TestTerminalCompression(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24), WithMaxScrollbackLines(100))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	if _, err := term.CompressionActivity(); err != nil {
		t.Fatal(err)
	}
	result, err := term.Compress(TerminalCompressionIncremental)
	if err != nil {
		t.Fatal(err)
	}
	switch result {
	case TerminalCompressionUnsupported,
		TerminalCompressionPending,
		TerminalCompressionComplete:
	default:
		t.Fatalf("unexpected compression result %d", result)
	}
}
