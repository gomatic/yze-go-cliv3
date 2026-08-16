package cliv3

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnquoteRejectsMalformedLiteral(t *testing.T) {
	t.Parallel()
	path, err := unquote("not-a-quoted-literal")
	require.ErrorIs(t, err, errUnquote)
	assert.Empty(t, path)
}

func TestUnquoteResolvesBothQuotingStyles(t *testing.T) {
	t.Parallel()
	for name, literal := range map[string]string{
		"interpreted": `"github.com/urfave/cli/v2"`,
		"raw":         "`github.com/urfave/cli/v2`",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path, err := unquote(importLiteral(literal))
			require.NoError(t, err)
			assert.Equal(t, pathV2, path)
		})
	}
}

func TestLegacyImport(t *testing.T) {
	t.Parallel()
	for literal, legacy := range map[string]bool{
		"not-a-quoted-literal":                false, // unquotable literals cannot match
		`"github.com/urfave/cli"`:             true,  // v1's module root
		`"github.com/urfave/cli/v2"`:          true,  // v2's module root
		"`github.com/urfave/cli/v2`":          true,  // v2, raw-string quoting
		`"gopkg.in/urfave/cli.v1"`:            true,  // v1 under its second module path
		`"github.com/urfave/cli/altsrc"`:      true,  // a v1 subpackage is v1 code
		`"github.com/urfave/cli/v2/altsrc"`:   true,  // a v2 subpackage is v2 code
		`"github.com/urfave/cli/v20"`:         false, // a major-version directory that is not v2
		`"github.com/urfave/cli/v3"`:          false, // the sanctioned version
		`"github.com/urfave/cli-alt"`:         false, // shares the v1 prefix, not the module
		`"github.com/urfave/climate"`:         false, // shares the bytes, not a segment boundary
		`"example.com/github.com/urfave/cli"`: false, // a legacy root is anchored at the start
	} {
		t.Run(literal, func(t *testing.T) {
			t.Parallel()
			_, got := legacyImport(importLiteral(literal))
			assert.Equal(t, legacy, got)
		})
	}
}

// TestEveryPackageOfEveryLegacyRootIsLegacy names the matcher's whole contract:
// which paths are legacy, and — for the ones that are — which subpackage they
// name, since the subpackage is what decides the replacement the diagnostic
// prescribes.
//
// v1's module path is the PREFIX of every later major, so the exclusion beneath
// it covers the major-version directories and NOTHING else. An earlier revision
// excluded the entire v1 namespace and two tests asserted that as intended,
// which silenced github.com/urfave/cli/altsrc — a real package of cli@v1.22.17
// that drags the v1 module into the build. Choosing a subpackage spelling
// acquired the silence and none of the property the exclusion exists for, so
// the tests encoded the hole rather than the contract and are corrected here.
//
// Each near-miss deviates from a flagged sibling in exactly one place, so each
// pins one side of one boundary rather than several at once.
func TestEveryPackageOfEveryLegacyRootIsLegacy(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		path   importPath
		why    string
		sub    subPackage
		legacy bool
	}{
		// The five legacy module roots.
		{path: pathV1, legacy: true, sub: "", why: "v1's module path itself"},
		{path: pathV2, legacy: true, sub: "", why: "v2's module path itself"},
		{path: pathGopkgV1, legacy: true, sub: "", why: "the same v1 code under its gopkg.in module path"},
		{path: pathOriginal, legacy: true, sub: "", why: "the name urfave/cli was renamed from, still served, byte-identical at v1.20.0"},
		{path: pathGopkgOriginal, legacy: true, sub: "", why: "and the original name under gopkg.in"},
		{path: pathOriginal + "/altsrc", legacy: true, sub: "altsrc", why: "the original name's subpackages too"},

		// Case folding. Import paths are case-sensitive to the toolchain, but
		// the hosts are not: before go.mod existed there is no declared module
		// path to disagree with, so github.com/Urfave/cli@v1.20.0 resolves,
		// builds and hands back the same *cli.App. Folding closes an unbounded
		// family of spellings with one decision.
		{path: "github.com/Urfave/cli", legacy: true, sub: "", why: "the host serves the same repository under either capitalisation"},
		{path: "github.com/urfave/Cli", legacy: true, sub: "", why: "and under any other"},
		{path: "GitHub.com/URFAVE/CLI/v2", legacy: true, sub: "", why: "the whole root folds, not one segment of it"},
		{path: "github.com/CodeGangsta/cli", legacy: true, sub: "", why: "the original name folds as well"},
		{path: "github.com/Urfave/cli/altsrc", legacy: true, sub: "altsrc", why: "a folded root still resolves its subpackage"},

		// Every package beneath a legacy root is legacy, and names its own
		// subpackage. A v2 subpackage must come back as "altsrc" and not as
		// v1's "v2/altsrc", which would prescribe the nonexistent cli-v2/v3;
		// what guarantees that is the major-version-directory exclusion applied
		// at every root, not the order the roots are listed in — measured, by
		// reversing the listing and getting byte-identical output.
		{path: pathV1 + "/altsrc", legacy: true, sub: "altsrc", why: "a v1 subpackage is v1 code"},
		{path: pathV2 + "/altsrc", legacy: true, sub: "altsrc", why: "a v2 subpackage is v2 code"},
		{path: pathGopkgV1 + "/altsrc", legacy: true, sub: "altsrc", why: "and so is a gopkg.in v1 subpackage"},
		{path: pathV1 + "/anything/deeper", legacy: true, sub: "anything/deeper", why: "the exclusion is the version directories, not the namespace"},

		// The exclusion beneath v1, at its boundary in both directions.
		{path: pathV1 + "/v3", legacy: false, why: "v3 is a major-version directory, not v1 code"},
		{path: pathV1 + "/v3/altsrc", legacy: false, why: "and neither is anything beneath one"},
		{path: pathV1 + "/v20", legacy: false, why: "a major-version directory that is not v2 either"},
		{path: pathV1 + "/version", legacy: true, sub: "version", why: "starts with v, is not v-plus-digits"},
		{path: pathV1 + "/v3x", legacy: true, sub: "v3x", why: "v-plus-digits plus one more character"},
		{path: pathV1 + "/v", legacy: true, sub: "v", why: "a bare v is not a version directory"},

		// The gopkg.in root, at its boundary. There is no gopkg.in v2:
		// gopkg.in/urfave/cli.v2 declares its module path as
		// github.com/urfave/cli/v2 and does not resolve under the gopkg.in path.
		{path: "gopkg.in/urfave/cli.v2", legacy: false, why: "gopkg.in serves no v2 of this module"},
		{path: "gopkg.in/urfave/cli.v10", legacy: false, why: "a different gopkg.in major"},
		{path: "gopkg.in/urfave/cli.v1x", legacy: false, why: "shares the bytes, not the module path"},

		// Paths that merely resemble a legacy root.
		{path: "github.com/urfave/climate", legacy: false, why: "shares the bytes without ending at a segment boundary"},
		{path: "github.com/urfave/cli-alt", legacy: false, why: "a hyphen is not a path separator"},
		{path: "github.com/urfave/cli-altsrc/v3", legacy: false, why: "the replacement this analyzer prescribes is not itself legacy"},
		{path: "github.com/urfave/cli-docs/v3", legacy: false, why: "nor is the other extracted module"},
		{path: "example.com/github.com/urfave/cli", legacy: false, why: "a legacy root is anchored at the start"},
		{path: "github.com/codegangsta/climate", legacy: false, why: "the original name gets the same boundary"},
		{path: "", legacy: false, why: "an empty path"},
	} {
		sub, legacy := legacyURFave(tc.path)
		assert.Equal(t, tc.legacy, legacy, "legacyURFave(%q) legacy: %s", tc.path, tc.why)
		if tc.legacy {
			assert.Equal(t, tc.sub, sub, "legacyURFave(%q) subpackage: %s", tc.path, tc.why)
		}
	}
}

// TestV3LineNamesAPackageThatExists names the remedy's claim. A diagnostic that
// prescribes a destination the author cannot reach leaves silencing the rule as
// the only move, so the replacement is not derived from what the marker
// suggests — it is the package urfave actually published.
//
// The reported subpackage set is closed and was MEASURED against the module
// cache on 2026-08-15, and the claim is about what is IMPORTABLE rather than
// about what directories exist — an earlier revision of this comment said "each
// carries exactly one Go subpackage", which is false and was caught by an
// adversarial pass. What was measured, counting non-test Go files per directory
// and reading each one's package clause:
//
//   - cli@v1.22.17 and gopkg.in/urfave/cli.v1@v1.20.0: altsrc, and nothing else.
//   - cli/v2@{v2.3.0,v2.27.7}: altsrc, plus internal/build,
//     internal/example-cli and internal/example-hello-world — all three
//     `package main` AND under internal/, so doubly unimportable.
//   - cli/v3@v3.10.1: examples/example-cli, examples/example-hello-world and
//     scripts — all `package main`, so the module offers no importable
//     subpackage, which is why a legacy subpackage cannot move to a path inside
//     it.
//
// Both destinations resolve on proxy.golang.org: github.com/urfave/cli/v3@v3.10.1
// and github.com/urfave/cli-altsrc/v3@v3.1.0 — while github.com/urfave/cli/v3/altsrc,
// which this analyzer used to prescribe for every reported path, 404s and has
// never existed.
func TestV3LineNamesAPackageThatExists(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		sub  subPackage
		want importPath
		why  string
	}{
		{sub: "", want: "github.com/urfave/cli/v3", why: "a module root moves to the v3 major of the same module"},
		{sub: "altsrc", want: "github.com/urfave/cli-altsrc/v3", why: "the only subpackage v1 or v2 ever carried, extracted at v3"},
		{sub: "docs", want: "github.com/urfave/cli-docs/v3", why: "the same extraction, published though never importable from v1/v2"},
		{sub: "altsrc/deeper", want: "github.com/urfave/cli-altsrc/v3", why: "the extracted module is named for the first segment"},
	} {
		assert.Equal(t, tc.want, v3Line(tc.sub), "v3Line(%q): %s", tc.sub, tc.why)
	}
}

// TestRootOrderDoesNotChangeTheVerdict names the invariant legacyRoots
// documents. v1's module path is a prefix of v2's, so a listing that tried v1
// first could plausibly claim ".../cli/v2/altsrc" as v1's subpackage "v2/altsrc"
// and prescribe the nonexistent cli-v2/v3. It does not, and the reason is the
// major-version-directory exclusion applied at every root rather than the order
// itself — which is exactly the kind of claim that is cheap to write down and
// wrong, so it is driven here against a reversed listing instead.
func TestRootOrderDoesNotChangeTheVerdict(t *testing.T) {
	t.Parallel()

	reversed := slices.Clone(legacyRoots())
	slices.Reverse(reversed)
	require.NotEqual(
		t,
		legacyRoots(),
		reversed,
		"the listing has more than one root, so reversing it is a real perturbation",
	)

	for _, path := range []importPath{
		pathV1, pathV2, pathGopkgV1, pathOriginal, pathGopkgOriginal,
		pathV1 + "/altsrc", pathV2 + "/altsrc", pathGopkgV1 + "/altsrc", pathOriginal + "/altsrc",
		pathV1 + "/v2", pathV1 + "/v2/altsrc", pathV1 + "/v3", pathV1 + "/v3/altsrc", pathV1 + "/v20",
		pathV1 + "/version", "github.com/Urfave/cli", "github.com/urfave/climate", pathV3, "",
	} {
		wantSub, wantLegacy := resolve(legacyRoots(), path)
		gotSub, gotLegacy := resolve(reversed, path)
		assert.Equal(t, wantLegacy, gotLegacy, "legacy verdict for %q moved with the root order", path)
		assert.Equal(t, wantSub, gotSub, "subpackage for %q moved with the root order", path)
	}
}

func TestIsMajorVersionDirIsVPlusDigits(t *testing.T) {
	t.Parallel()

	for segment, want := range map[pathSegment]bool{
		"v2":      true,
		"v3":      true,
		"v20":     true,
		"v0":      true,
		"v":       false,
		"v3x":     false,
		"version": false,
		"":        false,
		"2":       false,
		"V2":      false,
	} {
		assert.Equal(t, want, isMajorVersionDir(segment), "isMajorVersionDir(%q)", segment)
	}
}

// TestErrUnquoteIsAssertableEvenThoughItNeverEscapes names errUnquote's claim.
// It exists so unquote's failure contract can be asserted rather than assumed —
// an unparseable import literal cannot occur in type-checked code, so without a
// named sentinel the branch would be unreachable AND unverifiable, which is the
// combination that lets a "cannot happen" quietly become "happens and is
// mishandled".
func TestErrUnquoteIsAssertableEvenThoughItNeverEscapes(t *testing.T) {
	t.Parallel()

	_, err := unquote(importLiteral(`"github.com/urfave/cli/v2"`))
	require.NoError(t, err, "a well-formed literal unquotes")

	// Note: strconv.Unquote accepts a RUNE literal ('x'), so that is not a
	// failure case — only genuinely unquotable input is.
	for _, bad := range []string{``, `no quotes`, `"unterminated`, `"bad escape: \q"`} {
		_, err := unquote(importLiteral(bad))
		assert.ErrorIs(t, err, errUnquote, "%q is not a Go string literal", bad)
	}
}
