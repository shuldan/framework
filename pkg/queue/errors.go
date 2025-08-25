package queue

import "github.com/shuldan/framework/pkg/errors"

var newQueueCode = errors.WithPrefix("QUEUE")

var (
	ErrQueueClosed    = newQueueCode().New("cannot use closed queue")
	ErrInvalidJobType = newQueueCode().New("job type must be a non-nil pointer to struct")
	ErrMarshal        = newQueueCode().New("failed to marshal job for DLQ")
	ErrSendToDLQ      = newQueueCode().New("failed to send job to DLQ")
)
