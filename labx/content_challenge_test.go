package labx_test

import (
	"testing"
)

func TestChallenges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	testContent(t, "challenges")
}
