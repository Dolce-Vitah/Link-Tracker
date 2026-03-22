//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/testcontainers/testcontainers-go"
)

func requireTestcontainers(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	var (
		provider testcontainers.ContainerProvider
		err      error
	)
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				err = fmt.Errorf("panic while creating docker provider: %v", recovered)
			}
		}()
		provider, err = testcontainers.NewDockerProvider()
	}()
	if err != nil {
		t.Skipf("testcontainers are unavailable in this environment: %v", err)
		return
	}
	if provider != nil {
		_ = provider.Close()
	}
	_ = ctx
}
