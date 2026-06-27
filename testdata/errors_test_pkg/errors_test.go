//go:build !m0 && !m1

package errors_test_pkg_test

import (
	"errors"
	"testing"
)

func TestNewError(t *testing.T) {
	err := errors.New("oops")
	if err == nil {
		t.Fatal("expected non-nil error")
	}
}

func TestErrorMessage(t *testing.T) {
	err := errors.New("hello")
	got := err.Error()
	want := "hello"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
