package prompter

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func TestConfirmRetriesOnInvalidInput(t *testing.T) {
	input := "maybe\ny\n"

	ios := &iostreams.IOStreams{
		In:     io.NopCloser(strings.NewReader(input)),
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
	}
	forceTTY(ios)

	prompt := New(ios)
	got, err := prompt.Confirm("Proceed?", false)
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if !got {
		t.Fatalf("expected confirmation to be true after invalid input")
	}

	if !strings.Contains(ios.ErrOut.(*bytes.Buffer).String(), "Please respond") {
		t.Fatalf("expected error prompt after invalid input")
	}
}

func forceTTY(ios *iostreams.IOStreams) {
	setBoolField := func(name string) {
		field := reflect.ValueOf(ios).Elem().FieldByName(name)
		ptr := unsafe.Pointer(field.UnsafeAddr())
		reflect.NewAt(field.Type(), ptr).Elem().SetBool(true)
	}

	setBoolField("isStdinTTY")
	setBoolField("isStdoutTTY")
	setBoolField("isStderrTTY")
}
