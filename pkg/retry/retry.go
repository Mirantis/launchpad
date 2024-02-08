package retry

import (
	"fmt"
	"time"

	retry "github.com/avast/retry-go"
)

var Forever = false

func Infinite(task retry.RetryableFunc) error {
	for {
		if task() == nil {
			return nil
		}
	}
}

func Backoff(task retry.RetryableFunc, opts ...retry.Option) error {
	if Forever {
		return Infinite(task)
	}

	opts = append(
		[]retry.Option{
			retry.LastErrorOnly(true),
		},
		opts...,
	)

	if err := retry.Do(task, opts...); err != nil {
		return fmt.Errorf("retry with backoff: %w", err)
	}
	return nil
}

func Periodic(task retry.RetryableFunc, opts ...retry.Option) error {
	if Forever {
		return Infinite(task)
	}

	opts = append(
		[]retry.Option{
			retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
			retry.MaxJitter(time.Second * 2),
			retry.Delay(time.Second * 3),
			retry.Attempts(60),
			retry.LastErrorOnly(true),
		},
		opts...,
	)

	if err := retry.Do(task, opts...); err != nil {
		return fmt.Errorf("retry with periodic: %w", err)
	}
	return nil
}
