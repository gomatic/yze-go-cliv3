// Package testonly carries no legacy import of its own. Its test file does —
// the file kind is the second dimension every other fixture holds constant, and
// this analyzer is deliberately absent from the yze suite's source-only scope,
// so a rule keyed on the filename would silence every test file in the fleet
// with nothing in this suite going red.
package testonly

// Name identifies this fixture package.
func Name() string { return "testonly" }
