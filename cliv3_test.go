package cliv3_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/analysis/analysistest"

	cliv3 "github.com/gomatic/yze-cliv3"
)

func TestLegacyURFaveImportsAreReported(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), cliv3.Analyzer, "legacy", "good", "edges")
}

func TestRegistrationIsWellFormed(t *testing.T) {
	assert.NoError(t, cliv3.Registration.Validate())
	assert.Equal(t, "yze/cliv3", cliv3.Registration.RuleID())
	assert.Same(t, cliv3.Analyzer, cliv3.Registration.Analyzer)
}
