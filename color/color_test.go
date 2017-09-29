package color

import (
	"strconv"
	"testing"
)

func TestANSI(t *testing.T) {
	SetDefaultColor(ANSI)

	colorF := []func(string) string{
		Red, Green, Yellow, Blue, Magenta, Cyan,
	}
	for i, f := range colorF {
		if f("hello") != "\x1b[3"+strconv.Itoa(i+1)+"mhello\x1b[0m" {
			t.Errorf("%dth ansi color sequence wrong, got: %#v\n", i, f("hello"))
		}
	}
}

func TestNoColor(t *testing.T) {
	SetDefaultColor(NoColor)

	colorF := []func(string) string{
		Red, Green, Yellow, Blue, Magenta, Cyan,
	}
	for i, f := range colorF {
		if f("hello") != "hello" {
			t.Errorf("%dth no color sequence wrong, got: %#v\n", i, f("hello"))
		}
	}
}
