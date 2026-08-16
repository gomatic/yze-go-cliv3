package testonly

import (
	"testing"

	cliv1 "github.com/urfave/cli" // want `use github\.com/urfave/cli/v3;`
)

func TestApp(t *testing.T) {
	if (cliv1.App{}).Name != "" {
		t.Fatal("unexpected name")
	}
}
