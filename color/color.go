// Package color provides functions to generate colored text
// using ANSI color codes.
//
// For more information, refer to
// http://en.wikipedia.org/wiki/ANSI_escape_code#Colors
package color

// A Color specifies how to generated colored text.
type Color interface {
	Red(string) string
	Green(string) string
	Yellow(string) string
	Blue(string) string
	Magenta(string) string
	Cyan(string) string
}

var defaultColor Color = ANSI

// SetDefaultColor changes the color implementation used by global color
// function in this package.
func SetDefaultColor(c Color) {
	defaultColor = c
}

func Red(s string) string {
	return defaultColor.Red(s)
}

func Green(s string) string {
	return defaultColor.Green(s)
}

func Yellow(s string) string {
	return defaultColor.Yellow(s)
}

func Blue(s string) string {
	return defaultColor.Blue(s)
}

func Magenta(s string) string {
	return defaultColor.Magenta(s)
}

func Cyan(s string) string {
	return defaultColor.Cyan(s)
}

// ANSI implements the Color interface, it generates ANSI escape code to
// control text color.
var ANSI ansi

// NoColor implements the Color interface, it makes no transformation to the
// text.
var NoColor nocolor

type ansi struct{}

const ansiReset = "\x1b[0m"

func (c ansi) Red(s string) string {
	return "\x1b[31m" + s + ansiReset
}

func (c ansi) Green(s string) string {
	return "\x1b[32m" + s + ansiReset
}

func (c ansi) Yellow(s string) string {
	return "\x1b[33m" + s + ansiReset
}

func (c ansi) Blue(s string) string {
	return "\x1b[34m" + s + ansiReset
}

func (c ansi) Magenta(s string) string {
	return "\x1b[35m" + s + ansiReset
}

func (c ansi) Cyan(s string) string {
	return "\x1b[36m" + s + ansiReset
}

type nocolor struct{}

func (c nocolor) Red(s string) string {
	return s
}

func (c nocolor) Green(s string) string {
	return s
}

func (c nocolor) Yellow(s string) string {
	return s
}

func (c nocolor) Blue(s string) string {
	return s
}

func (c nocolor) Magenta(s string) string {
	return s
}

func (c nocolor) Cyan(s string) string {
	return s
}
