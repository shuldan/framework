package queue

import "time"

type Counter interface {
	IncProcessed(queue string, status ProcessedStatus)
	IncError(queue, handler string)
	IncRetry(queue string)
	IncDLQ(queue string)
	ObserveProcessingTime(queue string, duration time.Duration)
}

type ProcessedStatus string

const (
	StatusSuccess ProcessedStatus = "success"
	StatusError   ProcessedStatus = "error"
	StatusDLQ     ProcessedStatus = "dlq"
)

type NoOpCounter struct{}

func (NoOpCounter) IncProcessed(string, ProcessedStatus)        {}
func (NoOpCounter) IncError(string, string)                     {}
func (NoOpCounter) IncRetry(string)                             {}
func (NoOpCounter) IncDLQ(string)                               {}
func (NoOpCounter) ObserveProcessingTime(string, time.Duration) {}
