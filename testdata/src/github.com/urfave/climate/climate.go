// Package climate is an unrelated module whose path starts with the same bytes
// as urfave/cli without ending at a path segment boundary; it must NOT be
// flagged (false-positive guard).
package climate

// App is a stand-in exported type.
type App struct{ Name string }
