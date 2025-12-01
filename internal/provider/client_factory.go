// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/redis/go-redis/v9"
	glide "github.com/valkey-io/valkey-glide/go/v2"
	"github.com/valkey-io/valkey-glide/go/v2/config"
)

// createRedisClient creates a Redis client wrapper based on the provider configuration.
// It supports standalone, sentinel, and cluster modes.
// The REDIS_URL environment variable can override the configuration.
func createRedisClient(ctx context.Context, data RedisACLProviderModel, tlsConfig *tls.Config) (UniversalClient, error) {
	var client redis.UniversalClient

	// Override with REDIS_URL environment variable if set
	redisURL := os.Getenv("REDIS_URL")
	if redisURL != "" {
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			return nil, fmt.Errorf("invalid Redis URL: %w", err)
		}
		if tlsConfig != nil {
			opts.TLSConfig = tlsConfig
		}
		client = redis.NewClient(opts)
	} else if !data.Sentinel.IsNull() {
		// Sentinel configuration
		var sentinelModel SentinelModel
		diags := data.Sentinel.As(ctx, &sentinelModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("failed to parse sentinel configuration: %s", diags.Errors()[0].Summary())
		}

		var sentinelAddrs []string
		for _, addr := range sentinelModel.Addresses {
			sentinelAddrs = append(sentinelAddrs, addr.ValueString())
		}

		opts := &redis.FailoverOptions{
			MasterName:       sentinelModel.MasterName.ValueString(),
			SentinelAddrs:    sentinelAddrs,
			SentinelUsername: sentinelModel.Username.ValueString(),
			SentinelPassword: sentinelModel.Password.ValueString(),
			Username:         data.Username.ValueString(),
			Password:         data.Password.ValueString(),
			TLSConfig:        tlsConfig,
		}
		client = redis.NewFailoverClient(opts)
	} else if !data.Cluster.IsNull() {
		// Cluster configuration
		var clusterModel ClusterModel
		diags := data.Cluster.As(ctx, &clusterModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("failed to parse cluster configuration: %s", diags.Errors()[0].Summary())
		}

		var clusterAddrs []string
		for _, addr := range clusterModel.Addresses {
			clusterAddrs = append(clusterAddrs, addr.ValueString())
		}

		opts := &redis.ClusterOptions{
			Addrs:     clusterAddrs,
			Username:  data.Username.ValueString(),
			Password:  data.Password.ValueString(),
			TLSConfig: tlsConfig,
		}
		client = redis.NewClusterClient(opts)
	} else {
		// Single instance configuration
		address := "localhost:6379"
		if !data.Address.IsNull() {
			address = data.Address.ValueString()
		}

		opts := &redis.Options{
			Addr:      address,
			Username:  data.Username.ValueString(),
			Password:  data.Password.ValueString(),
			TLSConfig: tlsConfig,
		}
		client = redis.NewClient(opts)
	}

	// Wrap the redis client with our interface wrapper
	return NewRedisClientWrapper(client), nil
}

// createValkeyClient creates a Valkey client wrapper based on the provider configuration.
// It routes to the appropriate Valkey client creation function based on the configuration.
// Sentinel mode is not supported with Valkey.
func createValkeyClient(ctx context.Context, data RedisACLProviderModel, tlsConfig *tls.Config) (UniversalClient, error) {
	// Validate that Sentinel is not configured with Valkey
	if !data.Sentinel.IsNull() {
		return nil, fmt.Errorf("Valkey backend does not support Sentinel configuration")
	}

	// Route to cluster or standalone client creation
	if !data.Cluster.IsNull() {
		return createValkeyClusterClient(ctx, data, tlsConfig)
	}

	return createValkeyStandaloneClient(ctx, data, tlsConfig)
}

// createValkeyStandaloneClient creates a standalone Valkey client with address parsing,
// authentication, and TLS support.
func createValkeyStandaloneClient(_ context.Context, data RedisACLProviderModel, tlsConfig *tls.Config) (UniversalClient, error) {
	// Get address from configuration or use default
	address := "localhost:6379"
	if !data.Address.IsNull() {
		address = data.Address.ValueString()
	}

	// Parse host and port
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address format: %w", err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	// Build Valkey configuration
	clientConfig := config.NewClientConfiguration().
		WithAddress(&config.NodeAddress{Host: host, Port: port})

	// Add authentication if provided
	if !data.Username.IsNull() || !data.Password.IsNull() {
		clientConfig = clientConfig.WithCredentials(
			config.NewServerCredentials(
				data.Username.ValueString(),
				data.Password.ValueString(),
			),
		)
	}

	// Add TLS if configured
	if tlsConfig != nil {
		clientConfig = clientConfig.WithUseTLS(true)
	}

	// Create client
	client, err := glide.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Valkey client: %w", err)
	}

	return NewValkeyClientWrapper(client), nil
}

// createValkeyClusterClient creates a Valkey cluster client with cluster address parsing,
// authentication, and TLS support.
func createValkeyClusterClient(ctx context.Context, data RedisACLProviderModel, tlsConfig *tls.Config) (UniversalClient, error) {
	// Parse cluster configuration
	var clusterModel ClusterModel
	diags := data.Cluster.As(ctx, &clusterModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("failed to parse cluster configuration: %s", diags.Errors()[0].Summary())
	}

	// Build cluster configuration
	clusterConfig := config.NewClusterClientConfiguration()

	// Add addresses
	for _, addr := range clusterModel.Addresses {
		host, portStr, err := net.SplitHostPort(addr.ValueString())
		if err != nil {
			return nil, fmt.Errorf("invalid cluster address format: %w", err)
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid cluster port: %w", err)
		}

		clusterConfig = clusterConfig.WithAddress(&config.NodeAddress{Host: host, Port: port})
	}

	// Add authentication if provided
	if !data.Username.IsNull() || !data.Password.IsNull() {
		clusterConfig = clusterConfig.WithCredentials(
			config.NewServerCredentials(
				data.Username.ValueString(),
				data.Password.ValueString(),
			),
		)
	}

	// Add TLS if configured
	if tlsConfig != nil {
		clusterConfig = clusterConfig.WithUseTLS(true)
	}

	// Create cluster client
	client, err := glide.NewClusterClient(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Valkey cluster client: %w", err)
	}

	return NewValkeyClientWrapper(client), nil
}
