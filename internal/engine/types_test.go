package engine

import (
	"errors"
	"fmt"
	"testing"
)

func TestSourceErrorError(t *testing.T) {
	e := SourceError{
		Source: "my-source",
		Err:    fmt.Errorf("something went wrong"),
	}
	want := "my-source: something went wrong"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestSourceErrorUnwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	e := SourceError{
		Source: "src",
		Err:    inner,
	}
	if !errors.Is(e, inner) {
		t.Error("Unwrap should return inner error")
	}
}
