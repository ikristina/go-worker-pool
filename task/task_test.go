package task

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tests := map[string]struct {
		job           *Job
		expectErr     bool
		expectedValue string
	}{
		"Happy Path": {
			job:           &Job{ID: 1, URL: "https://google.com"},
			expectErr:     false,
			expectedValue: "200 OK",
		},
		"Invalid URL": {
			job:           &Job{ID: 2, URL: "not-a-url"},
			expectErr:     true,
			expectedValue: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			res := Run(ctx, tt.job)
			if tt.expectErr {
				if res.Err == nil {
					t.Errorf("%s: expected error but got nil", name)
				}
			} else {
				assert.NoError(t, res.Err, name)
				if res.Value != tt.expectedValue {
					t.Errorf("%s: got unexpected value: %v", name, res.Value)
				}
			}
		})
	}
}
