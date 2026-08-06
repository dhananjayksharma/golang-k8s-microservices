//go:build integration

package integration

import "testing"

func TestServicesAreReady(t *testing.T) {
	waitForHTTP(t, orderBaseURL+"/health/ready")
	waitForHTTP(t, inventoryBaseURL+"/healthz")
}
