// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	glide "github.com/valkey-io/valkey-glide/go/v2"
)

// UniversalClient defines the interface for both Redis and Valkey clients.
// This abstraction allows the provider to work with either backend seamlessly.
type UniversalClient interface {
	// ACLSetUser creates or updates an ACL user with the specified rules.
	ACLSetUser(ctx context.Context, username string, rules ...string) error

	// ACLDelUser deletes an ACL user.
	ACLDelUser(ctx context.Context, username string) error

	// Do executes a generic Redis/Valkey command.
	// This is used for commands like ACL GETUSER, ACL WHOAMI, ACL USERS, etc.
	Do(ctx context.Context, args ...interface{}) (interface{}, error)

	// Ping tests the connection to the server.
	Ping(ctx context.Context) (string, error)

	// Close closes the client connection.
	Close() error
}

// RedisClientWrapper wraps redis.UniversalClient to implement the UniversalClient interface.
// This allows the existing go-redis client to work with our abstraction layer.
type RedisClientWrapper struct {
	client redis.UniversalClient
}

// NewRedisClientWrapper creates a new RedisClientWrapper.
func NewRedisClientWrapper(client redis.UniversalClient) *RedisClientWrapper {
	return &RedisClientWrapper{
		client: client,
	}
}

// ACLSetUser creates or updates an ACL user with the specified rules.
func (w *RedisClientWrapper) ACLSetUser(ctx context.Context, username string, rules ...string) error {
	return w.client.ACLSetUser(ctx, username, rules...).Err()
}

// ACLDelUser deletes an ACL user.
func (w *RedisClientWrapper) ACLDelUser(ctx context.Context, username string) error {
	return w.client.ACLDelUser(ctx, username).Err()
}

// Do executes a generic Redis command.
func (w *RedisClientWrapper) Do(ctx context.Context, args ...interface{}) (interface{}, error) {
	return w.client.Do(ctx, args...).Result()
}

// Ping tests the connection to the Redis server.
func (w *RedisClientWrapper) Ping(ctx context.Context) (string, error) {
	return w.client.Ping(ctx).Result()
}

// Close closes the Redis client connection.
func (w *RedisClientWrapper) Close() error {
	return w.client.Close()
}

// ValkeyClientWrapper wraps valkey-glide client to implement the UniversalClient interface.
// This allows the Valkey Glide client to work with our abstraction layer.
// The client field can be either *glide.Client (standalone) or *glide.ClusterClient (cluster).
type ValkeyClientWrapper struct {
	client interface{} // Will be *glide.Client or *glide.ClusterClient
}

// NewValkeyClientWrapper creates a new ValkeyClientWrapper.
func NewValkeyClientWrapper(client interface{}) *ValkeyClientWrapper {
	return &ValkeyClientWrapper{
		client: client,
	}
}

// ACLSetUser creates or updates an ACL user with the specified rules.
func (w *ValkeyClientWrapper) ACLSetUser(ctx context.Context, username string, rules ...string) error {
	// Build ACL SETUSER command
	args := []string{"ACL", "SETUSER", username}
	args = append(args, rules...)

	// Execute using CustomCommand
	_, err := w.executeCommand(ctx, args...)
	return err
}

// ACLDelUser deletes an ACL user.
func (w *ValkeyClientWrapper) ACLDelUser(ctx context.Context, username string) error {
	_, err := w.executeCommand(ctx, "ACL", "DELUSER", username)
	return err
}

// Do executes a generic Valkey command.
func (w *ValkeyClientWrapper) Do(ctx context.Context, args ...interface{}) (interface{}, error) {
	// Convert args to string slice
	strArgs := make([]string, len(args))
	for i, arg := range args {
		strArgs[i] = fmt.Sprint(arg)
	}
	return w.executeCommand(ctx, strArgs...)
}

// Ping tests the connection to the Valkey server.
func (w *ValkeyClientWrapper) Ping(ctx context.Context) (string, error) {
	// Use type assertion to call appropriate Ping method
	switch client := w.client.(type) {
	case *glide.Client:
		return client.Ping(ctx)
	case *glide.ClusterClient:
		return client.Ping(ctx)
	default:
		return "", fmt.Errorf("unsupported client type: %T", w.client)
	}
}

// Close closes the Valkey client connection.
func (w *ValkeyClientWrapper) Close() error {
	// Use type assertion to call appropriate Close method
	switch client := w.client.(type) {
	case *glide.Client:
		client.Close()
		return nil
	case *glide.ClusterClient:
		client.Close()
		return nil
	default:
		return fmt.Errorf("unsupported client type: %T", w.client)
	}
}

// executeCommand handles command execution for both standalone and cluster clients.
// This is a helper method that abstracts the differences between the two client types.
func (w *ValkeyClientWrapper) executeCommand(ctx context.Context, args ...string) (interface{}, error) {
	switch client := w.client.(type) {
	case *glide.Client:
		return client.CustomCommand(ctx, args)
	case *glide.ClusterClient:
		return client.CustomCommand(ctx, args)
	default:
		return nil, fmt.Errorf("unsupported client type: %T", w.client)
	}
}
