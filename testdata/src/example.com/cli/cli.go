// Package cli is an unrelated module whose path merely contains "cli";
// it must NOT be flagged (false-positive guard).
package cli

// App is a stand-in exported type.
type App struct{ Name string }
