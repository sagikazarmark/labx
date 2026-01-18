package labx_test

import (
	"testing"
)

func TestTrainings(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	testContent(t, "trainings")
}
