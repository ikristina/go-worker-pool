package task

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// Result to output the result or error of the worker's job
type Result struct {
	ID int
	Value string
	Err error
}

// Job is an input to a worker
type Job struct {
	ID int
	URL string
}

// Run to act on a task
func Run(ctx context.Context, j *Job) *Result {
	if j == nil {
		return nil
	}

	r := &Result{
		ID: j.ID,
	}

	// fetch URL content
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, j.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		r.Err = fmt.Errorf("fetch failed: %w", err)
		return r
	}
	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		r.Err = fmt.Errorf("failed reading content: %w", err)
		return r
	}

	r.Value = resp.Status
	return r
}
