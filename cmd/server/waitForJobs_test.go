package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestWaitForJobsReturnsImmediatelyWithoutJobs(t *testing.T) {
	var jobs sync.WaitGroup
	if err := waitForJobs(context.Background(), &jobs); err != nil {
		t.Fatalf("waitForJobs: %v", err)
	}
}

func TestWaitForJobsWaitsUntilJobFinishes(t *testing.T) {
	var jobs sync.WaitGroup
	jobs.Add(1)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		result <- waitForJobs(ctx, &jobs)
	}()

	select {
	case err := <-result:
		t.Fatalf("waitForJobs returned before Done: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	jobs.Done()
	if err := <-result; err != nil {
		t.Fatalf("waitForJobs after Done: %v", err)
	}
}

func TestWaitForJobsReturnsWhenContextExpires(t *testing.T) {
	var jobs sync.WaitGroup
	jobs.Add(1)
	defer jobs.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := waitForJobs(ctx, &jobs); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waitForJobs error = %v, want deadline exceeded", err)
	}
}
