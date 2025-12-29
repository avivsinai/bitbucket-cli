package iostreams

import (
	"bytes"
	"testing"
)

func TestAlternateScreenBuffer(t *testing.T) {
	t.Parallel()

	t.Run("StartAlternateScreenBuffer writes escape sequence when TTY", func(t *testing.T) {
		buf := &bytes.Buffer{}
		ios := &IOStreams{
			Out:         buf,
			isStdoutTTY: true,
		}
		ios.StartAlternateScreenBuffer()
		if buf.String() != enterAltScreen {
			t.Errorf("expected %q, got %q", enterAltScreen, buf.String())
		}
	})

	t.Run("StartAlternateScreenBuffer no-op when not TTY", func(t *testing.T) {
		buf := &bytes.Buffer{}
		ios := &IOStreams{
			Out:         buf,
			isStdoutTTY: false,
		}
		ios.StartAlternateScreenBuffer()
		if buf.Len() != 0 {
			t.Errorf("expected empty buffer for non-TTY, got %q", buf.String())
		}
	})

	t.Run("StopAlternateScreenBuffer writes escape sequence when TTY", func(t *testing.T) {
		buf := &bytes.Buffer{}
		ios := &IOStreams{
			Out:         buf,
			isStdoutTTY: true,
		}
		ios.StopAlternateScreenBuffer()
		if buf.String() != exitAltScreen {
			t.Errorf("expected %q, got %q", exitAltScreen, buf.String())
		}
	})

	t.Run("StopAlternateScreenBuffer no-op when not TTY", func(t *testing.T) {
		buf := &bytes.Buffer{}
		ios := &IOStreams{
			Out:         buf,
			isStdoutTTY: false,
		}
		ios.StopAlternateScreenBuffer()
		if buf.Len() != 0 {
			t.Errorf("expected empty buffer for non-TTY, got %q", buf.String())
		}
	})

	t.Run("ClearScreen writes escape sequence when TTY", func(t *testing.T) {
		buf := &bytes.Buffer{}
		ios := &IOStreams{
			Out:         buf,
			isStdoutTTY: true,
		}
		ios.ClearScreen()
		if buf.String() != clearScreen {
			t.Errorf("expected %q, got %q", clearScreen, buf.String())
		}
	})

	t.Run("ClearScreen no-op when not TTY", func(t *testing.T) {
		buf := &bytes.Buffer{}
		ios := &IOStreams{
			Out:         buf,
			isStdoutTTY: false,
		}
		ios.ClearScreen()
		if buf.Len() != 0 {
			t.Errorf("expected empty buffer for non-TTY, got %q", buf.String())
		}
	})

	t.Run("nil IOStreams handled gracefully", func(t *testing.T) {
		var ios *IOStreams
		// Should not panic
		ios.StartAlternateScreenBuffer()
		ios.StopAlternateScreenBuffer()
		ios.ClearScreen()
	})
}
