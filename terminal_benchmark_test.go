package libghostty

import (
	"bytes"
	"testing"
)

const terminalVTWriteBenchmarkBytes = 64 * 1024

// BenchmarkTerminalVTWrite measures the full Go-to-libghostty VT write path.
// The text and sparse-mode cases model normal terminal traffic without
// allowing screen or scrollback state to grow between iterations. The dense
// mode cases deliberately alternate every sequence to make the transition
// check and optional C-to-Go callback visible as a worst-case bound.
func BenchmarkTerminalVTWrite(b *testing.B) {
	textUnit := []byte("0123456789abcdef\r")
	text := bytes.Repeat(textUnit, terminalVTWriteBenchmarkBytes/len(textUnit))

	const modeSet = "\x1b[?1004h"
	const modeReset = "\x1b[?1004l"
	sparseModes := make([]byte, 0, len(text)+len(modeSet)+len(modeReset))
	sparseModes = append(sparseModes, modeSet...)
	sparseModes = append(sparseModes, text...)
	sparseModes = append(sparseModes, modeReset...)

	denseUnit := []byte(modeSet + modeReset)
	denseModes := bytes.Repeat(
		denseUnit,
		terminalVTWriteBenchmarkBytes/len(denseUnit),
	)
	denseTransitions := 2 * (terminalVTWriteBenchmarkBytes / len(denseUnit))

	b.Run("Text", func(b *testing.B) {
		benchmarkTerminalVTWrite(b, text)
	})
	b.Run("SparseModes/NoCallback", func(b *testing.B) {
		benchmarkTerminalVTWrite(b, sparseModes)
	})
	b.Run("SparseModes/Callback", func(b *testing.B) {
		callbacks := 0
		benchmarkTerminalVTWrite(
			b,
			sparseModes,
			WithModeChanged(func(*Terminal, Mode, bool) {
				callbacks++
			}),
		)

		if want := b.N * 2; callbacks != want {
			b.Fatalf("mode callbacks = %d, want %d", callbacks, want)
		}
	})
	b.Run("DenseModes/NoCallback", func(b *testing.B) {
		benchmarkTerminalVTWrite(b, denseModes)
	})
	b.Run("DenseModes/Callback", func(b *testing.B) {
		callbacks := 0
		benchmarkTerminalVTWrite(
			b,
			denseModes,
			WithModeChanged(func(*Terminal, Mode, bool) {
				callbacks++
			}),
		)

		if want := b.N * denseTransitions; callbacks != want {
			b.Fatalf("mode callbacks = %d, want %d", callbacks, want)
		}
	})
}

func benchmarkTerminalVTWrite(
	b *testing.B,
	data []byte,
	opts ...TerminalOption,
) {
	b.Helper()
	opts = append([]TerminalOption{WithSize(80, 24)}, opts...)
	term, err := NewTerminal(opts...)
	if err != nil {
		b.Fatal(err)
	}
	defer term.Close()

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for b.Loop() {
		term.VTWrite(data)
	}
	b.StopTimer()
}
