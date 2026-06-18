package libghostty

import "testing"

func TestRenderStateNewClose(t *testing.T) {
	rs, err := NewRenderState()
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
}

func TestRenderStateUpdate(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	rs, err := NewRenderState()
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()

	if err := rs.Update(term); err != nil {
		t.Fatal(err)
	}

	cols, err := rs.Cols()
	if err != nil {
		t.Fatal(err)
	}
	if cols != 80 {
		t.Fatalf("expected 80 cols, got %d", cols)
	}

	rows, err := rs.Rows()
	if err != nil {
		t.Fatal(err)
	}
	if rows != 24 {
		t.Fatalf("expected 24 rows, got %d", rows)
	}
}

func TestRenderStateDirty(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	rs, err := NewRenderState()
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()

	// First update should be dirty (full).
	if err := rs.Update(term); err != nil {
		t.Fatal(err)
	}
	dirty, err := rs.Dirty()
	if err != nil {
		t.Fatal(err)
	}
	if dirty != RenderStateDirtyFull {
		t.Fatalf("expected full dirty after first update, got %d", dirty)
	}

	// Clear dirty and verify.
	if err := rs.SetDirty(RenderStateDirtyFalse); err != nil {
		t.Fatal(err)
	}
	dirty, _ = rs.Dirty()
	if dirty != RenderStateDirtyFalse {
		t.Fatalf("expected not dirty after clearing, got %d", dirty)
	}
}

func TestRenderStateColors(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	// Set colors so we have known values.
	red := &ColorRGB{R: 255, G: 0, B: 0}
	blue := &ColorRGB{R: 0, G: 0, B: 255}
	if err := term.SetColorForeground(red); err != nil {
		t.Fatal(err)
	}
	if err := term.SetColorBackground(blue); err != nil {
		t.Fatal(err)
	}

	rs, err := NewRenderState()
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()

	if err := rs.Update(term); err != nil {
		t.Fatal(err)
	}

	fg, err := rs.ColorForeground()
	if err != nil {
		t.Fatal(err)
	}
	if fg != *red {
		t.Fatalf("expected fg %+v, got %+v", *red, fg)
	}

	bg, err := rs.ColorBackground()
	if err != nil {
		t.Fatal(err)
	}
	if bg != *blue {
		t.Fatalf("expected bg %+v, got %+v", *blue, bg)
	}

	// Bulk colors API.
	colors, err := rs.Colors()
	if err != nil {
		t.Fatal(err)
	}
	if colors.Foreground != *red {
		t.Fatalf("expected colors.Foreground %+v, got %+v", *red, colors.Foreground)
	}
	if colors.Background != *blue {
		t.Fatalf("expected colors.Background %+v, got %+v", *blue, colors.Background)
	}
}

func TestRenderStateDefaultColors(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	rs, err := NewRenderState()
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()

	if err := rs.Update(term); err != nil {
		t.Fatal(err)
	}

	defaults, err := rs.DefaultColors()
	if err != nil {
		t.Fatal(err)
	}
	if defaults.ForegroundHasValue || defaults.BackgroundHasValue || defaults.CursorHasValue {
		t.Fatalf("expected unset defaults, got %+v", defaults)
	}

	red := &ColorRGB{R: 255, G: 0, B: 0}
	blue := &ColorRGB{R: 0, G: 0, B: 255}
	green := &ColorRGB{R: 0, G: 255, B: 0}
	if err := term.SetColorForeground(red); err != nil {
		t.Fatal(err)
	}
	if err := term.SetColorBackground(blue); err != nil {
		t.Fatal(err)
	}
	if err := term.SetColorCursor(green); err != nil {
		t.Fatal(err)
	}
	if err := rs.Update(term); err != nil {
		t.Fatal(err)
	}

	defaults, err = rs.DefaultColors()
	if err != nil {
		t.Fatal(err)
	}
	if !defaults.ForegroundHasValue || defaults.Foreground != *red {
		t.Fatalf("expected foreground %+v, got %+v", *red, defaults)
	}
	if !defaults.BackgroundHasValue || defaults.Background != *blue {
		t.Fatalf("expected background %+v, got %+v", *blue, defaults)
	}
	if !defaults.CursorHasValue || defaults.Cursor != *green {
		t.Fatalf("expected cursor %+v, got %+v", *green, defaults)
	}
}

func TestRenderStateCursor(t *testing.T) {
	term, err := NewTerminal(WithSize(80, 24))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	rs, err := NewRenderState()
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()

	if err := rs.Update(term); err != nil {
		t.Fatal(err)
	}

	visible, err := rs.CursorVisible()
	if err != nil {
		t.Fatal(err)
	}
	if !visible {
		t.Fatal("expected cursor visible by default")
	}

	hasViewport, err := rs.CursorViewportHasValue()
	if err != nil {
		t.Fatal(err)
	}
	if !hasViewport {
		t.Fatal("expected cursor in viewport by default")
	}

	x, err := rs.CursorViewportX()
	if err != nil {
		t.Fatal(err)
	}
	y, err := rs.CursorViewportY()
	if err != nil {
		t.Fatal(err)
	}
	if x != 0 || y != 0 {
		t.Fatalf("expected cursor at 0,0, got %d,%d", x, y)
	}
}
