package util

import (
	"math"
	mathrand "math/rand/v2"
	"time"
)

func RetryWithBackoff(work func() error, maxRetries int) error {
	var retries float64

retry:
	err := work()
	if err == nil {
		return nil
	}

	if retries >= float64(maxRetries) {
		return err
	}

	toSleep := math.Min(30.0, math.Pow(2, retries))

	// Add jitter
	toSleep = mathrand.Float64() * toSleep

	// Sleep
	time.Sleep(time.Second * time.Duration(toSleep))

	retries += 1.0
	goto retry
}
