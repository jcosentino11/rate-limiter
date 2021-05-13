package ratelimiter

import (
	"testing"
	"time"
)

func TestHello(t *testing.T) {
	if HelloWorld() == "" {
		t.Fail()
	}
}

func TestHelloIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	time.Sleep(1 * time.Second)
	t.Fail()
}
