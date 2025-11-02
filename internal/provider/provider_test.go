// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// connect.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"redisacl": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// Verify testcontainers can run (Docker daemon available)
	ctx := context.Background()
	
	// Check if Redis container is running
	if redisHost == "" || redisPort == "" {
		t.Fatal("Redis container not started. Ensure TestMain has been called to start the container.")
	}
	
	// Verify Redis container is accessible
	if err := WaitForRedisReady(ctx); err != nil {
		t.Fatalf("Redis container not accessible: %v", err)
	}
	
	// Verify REDIS_URL is set (should be set by StartRedisContainer)
	if v := os.Getenv("REDIS_URL"); v == "" {
		t.Fatal("REDIS_URL must be set for acceptance tests")
	}
	
	// Clean up any existing test users before each test
	if err := CleanupRedisUsers(ctx); err != nil {
		t.Logf("Warning: failed to cleanup Redis users: %v", err)
	}
}