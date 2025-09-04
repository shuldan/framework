package queue

import (
	"testing"
	"time"
)

func TestNoOpCounter(t *testing.T) {
	t.Parallel()
	counter := NoOpCounter{}

	counter.IncProcessed("test", StatusSuccess)
	counter.IncError("test", "handler")
	counter.IncRetry("test")
	counter.IncDLQ("test")
	counter.ObserveProcessingTime("test", time.Second)
}
