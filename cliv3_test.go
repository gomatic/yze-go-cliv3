package cliv3_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	goyze "github.com/gomatic/go-yze"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"

	cliv3 "github.com/gomatic/yze-go-cliv3"
)

// message is the diagnostic's contract, written from the package doc comment
// rather than from a run: the first clause names the package that REPLACES the
// reported one, and the rest says which standard forbids it.
func message(replacement string) string {
	return "use " + replacement + "; the legacy urfave/cli v1/v2 import path is forbidden by the gomatic CLI standard"
}

// fixtures names every fixture package. The two dimensions that decide where
// this rule reaches are the package clause and the file kind, so both vary
// here: `library` is not package main, and `testonly`'s import sits in a
// _test.go file. Holding either constant makes a rule keyed on it invisible to
// this suite while it silences most of the fleet.
var fixtures = []string{"legacy", "good", "edges", "library", "testonly"}

func TestLegacyURFaveImportsAreReported(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), cliv3.Analyzer, fixtures...)
}

// TestDiagnosticNamesTheReplacementAtThePathLiteral owns the two parts of the
// standard a `want` comment cannot see: the message BODY (analysistest matches
// a regexp, and every want here is deliberately truncated to the clause that
// varies) and the COLUMN (analysistest matches by line alone, so reporting the
// import spec rather than its path literal is invisible to it).
//
// The expected columns are not transcribed from a run. Each is resolved from
// the fixture source by finding the import path literal on the reported line,
// so the claim stays "the diagnostic points at the path literal" rather than
// "the diagnostic points where it pointed last time".
func TestDiagnosticNamesTheReplacementAtThePathLiteral(t *testing.T) {
	want := map[string]string{
		`"github.com/urfave/cli"`:           "github.com/urfave/cli/v3",
		`"github.com/urfave/cli/v2"`:        "github.com/urfave/cli/v3",
		"`github.com/urfave/cli/v2`":        "github.com/urfave/cli/v3",
		`"gopkg.in/urfave/cli.v1"`:          "github.com/urfave/cli/v3",
		`"github.com/urfave/cli/altsrc"`:    "github.com/urfave/cli-altsrc/v3",
		`"github.com/urfave/cli/v2/altsrc"`: "github.com/urfave/cli-altsrc/v3",
		`"gopkg.in/urfave/cli.v1/altsrc"`:   "github.com/urfave/cli-altsrc/v3",
	}

	results := analysistest.Run(t, analysistest.TestData(), cliv3.Analyzer, "edges")
	require.Len(t, results, 1, "one fixture package, one action")
	pass := results[0].Pass
	require.NotNil(t, pass, "the action ran")

	source := readLines(t, filepath.Join(analysistest.TestData(), "src", "edges", "edges.go"))

	seen := map[string]int{}
	for _, diagnostic := range results[0].Diagnostics {
		position := pass.Fset.Position(diagnostic.Pos)
		require.LessOrEqual(t, position.Line, len(source), "reported line is in the fixture")
		tail := source[position.Line-1][position.Column-1:]

		literal, replacement, matched := startsWithLiteral(tail, want)
		require.True(
			t,
			matched,
			"the diagnostic starts at an import path literal; at %d:%d it starts at %q",
			position.Line,
			position.Column,
			tail,
		)
		assert.Equal(t, message(replacement), diagnostic.Message, "message for %s", literal)
		seen[literal]++
	}

	expected := map[string]int{}
	for literal := range want {
		expected[literal] = 1
	}
	assert.Equal(t, expected, seen, "every legacy literal in the fixture is reported exactly once")
}

// startsWithLiteral resolves the import path literal a diagnostic points at,
// and the replacement its message must name.
func startsWithLiteral(tail string, want map[string]string) (string, string, bool) {
	for literal, replacement := range want {
		if strings.HasPrefix(tail, literal) {
			return literal, replacement, true
		}
	}
	return "", "", false
}

// readLines reads a fixture file as its lines, so a reported position can be
// resolved back to the source it names.
func readLines(t *testing.T, path string) []string {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return strings.Split(string(content), "\n")
}

func TestRegistrationIsWellFormed(t *testing.T) {
	assert.NoError(t, cliv3.Registration.Validate())
	assert.Equal(t, "yze/cliv3", cliv3.Registration.RuleID())
	assert.Same(t, cliv3.Analyzer, cliv3.Registration.Analyzer)
}

// TestRegistrationCategoryIsCLI pins the category. It is not cosmetic: `yze
// --category` selects analyzers by it, so a category nothing asserts is a
// fleet-wide selection change that no test and no corpus case can see.
func TestRegistrationCategoryIsCLI(t *testing.T) {
	assert.Equal(t, []goyze.Category{"cli"}, cliv3.Registration.Categories)
}
