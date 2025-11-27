// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/valkey"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	redisContainer  testcontainers.Container
	redisHost       string
	redisPort       string
	valkeyContainer *valkey.ValkeyContainer
	valkeyHost      string
	valkeyPort      string
)

// StartRedisContainer starts a Redis container with ACL support for testing
func StartRedisContainer(ctx context.Context) error {
	if redisContainer != nil {
		return nil // Already started
	}

	req := testcontainers.ContainerRequest{
		Image:        "redis:7.4.7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		Cmd:          []string{"redis-server", "--requirepass", "testpass"},
		WaitingFor: wait.ForAll(
			wait.ForLog("Ready to accept connections"),
			wait.ForListeningPort("6379/tcp"),
		).WithDeadline(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("failed to start Redis container: %w", err)
	}

	redisContainer = container

	// Get the mapped port
	mappedPort, err := container.MappedPort(ctx, "6379")
	if err != nil {
		return fmt.Errorf("failed to get mapped port: %w", err)
	}

	// Get the host
	host, err := container.Host(ctx)
	if err != nil {
		return fmt.Errorf("failed to get container host: %w", err)
	}

	redisHost = host
	redisPort = mappedPort.Port()

	// Wait for Redis to be ready
	if err := WaitForRedisReady(ctx); err != nil {
		return fmt.Errorf("redis container not ready: %w", err)
	}

	// Set environment variables for tests
	_ = os.Setenv("REDIS_HOST", redisHost)
	_ = os.Setenv("REDIS_PORT", redisPort)
	_ = os.Setenv("REDIS_URL", GetRedisConnectionString())

	log.Printf("Redis container started at %s:%s", redisHost, redisPort)
	return nil
}

// StopRedisContainer stops and removes the Redis container
func StopRedisContainer(ctx context.Context) error {
	if redisContainer == nil {
		return nil
	}

	err := redisContainer.Terminate(ctx)
	redisContainer = nil
	redisHost = ""
	redisPort = ""

	// Clean up environment variables
	_ = os.Unsetenv("REDIS_HOST")
	_ = os.Unsetenv("REDIS_PORT")
	_ = os.Unsetenv("REDIS_URL")

	return err
}

// GetRedisConnectionString returns the connection string for the test Redis instance
func GetRedisConnectionString() string {
	if redisHost == "" || redisPort == "" {
		return ""
	}
	return fmt.Sprintf("redis://default:testpass@%s:%s/0", redisHost, redisPort)
}

// WaitForRedisReady waits for Redis to be ready to accept connections
func WaitForRedisReady(ctx context.Context) error {
	if redisHost == "" || redisPort == "" {
		return fmt.Errorf("redis container not started")
	}

	port, err := strconv.Atoi(redisPort)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisHost, port),
		Password: "testpass",
		DB:       0,
	})
	defer func() { _ = client.Close() }()

	// Try to ping Redis with timeout
	timeout := 30 * time.Second
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		_, err := client.Ping(ctx).Result()
		if err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("redis not ready after %v", timeout)
}

// CleanupRedisUsers removes all test users from Redis
func CleanupRedisUsers(ctx context.Context) error {
	if redisHost == "" || redisPort == "" {
		return nil
	}

	port, err := strconv.Atoi(redisPort)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisHost, port),
		Password: "testpass",
		DB:       0,
	})
	defer func() { _ = client.Close() }()

	// Get all users
	users, err := client.Do(ctx, "ACL", "USERS").StringSlice()
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	// Delete test users (keep default user)
	for _, user := range users {
		if user != "default" {
			err := client.ACLDelUser(ctx, user).Err()
			if err != nil {
				log.Printf("Warning: failed to delete user %s: %v", user, err)
			}
		}
	}

	return nil
}

// CreateTestUser creates a test user in Redis for testing
func CreateTestUser(ctx context.Context, username, password string) error {
	if redisHost == "" || redisPort == "" {
		return fmt.Errorf("redis container not started")
	}

	port, err := strconv.Atoi(redisPort)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisHost, port),
		Password: "testpass",
		DB:       0,
	})
	defer func() { _ = client.Close() }()

	rules := []string{"reset", "on", ">" + password, "~*", "&*", "+@all"}
	return client.ACLSetUser(ctx, username, rules...).Err()
}

// UserExists checks if a user exists in Redis
func UserExists(ctx context.Context, username string) (bool, error) {
	if redisHost == "" || redisPort == "" {
		return false, fmt.Errorf("redis container not started")
	}

	port, err := strconv.Atoi(redisPort)
	if err != nil {
		return false, fmt.Errorf("invalid port: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisHost, port),
		Password: "testpass",
		DB:       0,
	})
	defer func() { _ = client.Close() }()

	_, err = client.Do(ctx, "ACL", "GETUSER", username).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateTestUserWithSelectors creates a test user with selectors in Redis for testing
func CreateTestUserWithSelectors(ctx context.Context, username, password string) error {
	if redisHost == "" || redisPort == "" {
		return fmt.Errorf("redis container not started")
	}

	port, err := strconv.Atoi(redisPort)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisHost, port),
		Password: "testpass",
		DB:       0,
	})
	defer func() { _ = client.Close() }()

	rules := []string{"reset", "on", ">" + password, "~*", "&*", "+@all", "(~key* +get)", "(~data* +set)"}
	return client.ACLSetUser(ctx, username, rules...).Err()
}

// ModifyUserInRedis modifies a user in Redis with the given rules
func ModifyUserInRedis(ctx context.Context, username string, rules []string) error {
	if redisHost == "" || redisPort == "" {
		return fmt.Errorf("redis container not started")
	}

	port, err := strconv.Atoi(redisPort)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisHost, port),
		Password: "testpass",
		DB:       0,
	})
	defer func() { _ = client.Close() }()

	return client.ACLSetUser(ctx, username, rules...).Err()
}

// StartValkeyContainer starts a Valkey container with ACL support for testing
func StartValkeyContainer(ctx context.Context) error {
	if valkeyContainer != nil {
		return nil // Already started
	}

	container, err := valkey.Run(ctx,
		"valkey/valkey:8.0.1",
		testcontainers.WithCmd("valkey-server", "--requirepass", "testpass"),
	)
	if err != nil {
		return fmt.Errorf("failed to start Valkey container: %w", err)
	}

	valkeyContainer = container

	// Get the host
	host, err := container.Host(ctx)
	if err != nil {
		return fmt.Errorf("failed to get container host: %w", err)
	}

	// Get the mapped port
	mappedPort, err := container.MappedPort(ctx, "6379")
	if err != nil {
		return fmt.Errorf("failed to get mapped port: %w", err)
	}

	valkeyHost = host
	valkeyPort = mappedPort.Port()

	// Set environment variables for tests
	_ = os.Setenv("VALKEY_HOST", valkeyHost)
	_ = os.Setenv("VALKEY_PORT", valkeyPort)
	_ = os.Setenv("VALKEY_URL", GetValkeyConnectionString())

	log.Printf("Valkey container started at %s:%s", valkeyHost, valkeyPort)
	return nil
}

// StopValkeyContainer stops and removes the Valkey container
func StopValkeyContainer(ctx context.Context) error {
	if valkeyContainer == nil {
		return nil
	}

	err := valkeyContainer.Terminate(ctx)
	valkeyContainer = nil
	valkeyHost = ""
	valkeyPort = ""

	// Clean up environment variables
	_ = os.Unsetenv("VALKEY_HOST")
	_ = os.Unsetenv("VALKEY_PORT")
	_ = os.Unsetenv("VALKEY_URL")

	return err
}

// GetValkeyConnectionString returns the connection string for the test Valkey instance
func GetValkeyConnectionString() string {
	if valkeyHost == "" || valkeyPort == "" {
		return ""
	}
	return fmt.Sprintf("valkey://default:testpass@%s:%s/0", valkeyHost, valkeyPort)
}

// WaitForValkeyReady waits for Valkey to be ready to accept connections
func WaitForValkeyReady(ctx context.Context) error {
	if valkeyHost == "" || valkeyPort == "" {
		return fmt.Errorf("valkey container not started")
	}

	port, err := strconv.Atoi(valkeyPort)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", valkeyHost, port),
		Password: "testpass",
		DB:       0,
	})
	defer func() { _ = client.Close() }()

	// Try to ping Valkey with timeout
	timeout := 30 * time.Second
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		_, err := client.Ping(ctx).Result()
		if err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("valkey not ready after %v", timeout)
}

// CleanupValkeyUsers removes all test users from Valkey
func CleanupValkeyUsers(ctx context.Context) error {
	if valkeyHost == "" || valkeyPort == "" {
		return nil
	}

	port, err := strconv.Atoi(valkeyPort)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", valkeyHost, port),
		Password: "testpass",
		DB:       0,
	})
	defer func() { _ = client.Close() }()

	// Get all users
	users, err := client.Do(ctx, "ACL", "USERS").StringSlice()
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	// Delete test users (keep default user)
	for _, user := range users {
		if user != "default" {
			err := client.ACLDelUser(ctx, user).Err()
			if err != nil {
				log.Printf("Warning: failed to delete user %s: %v", user, err)
			}
		}
	}

	return nil
}

// CreateTestUserInValkey creates a test user in Valkey for testing
func CreateTestUserInValkey(ctx context.Context, username, password string) error {
	if valkeyHost == "" || valkeyPort == "" {
		return fmt.Errorf("valkey container not started")
	}

	port, err := strconv.Atoi(valkeyPort)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", valkeyHost, port),
		Password: "testpass",
		DB:       0,
	})
	defer func() { _ = client.Close() }()

	rules := []string{"reset", "on", ">" + password, "~*", "&*", "+@all"}
	return client.ACLSetUser(ctx, username, rules...).Err()
}

// UserExistsInValkey checks if a user exists in Valkey
func UserExistsInValkey(ctx context.Context, username string) (bool, error) {
	if valkeyHost == "" || valkeyPort == "" {
		return false, fmt.Errorf("valkey container not started")
	}

	port, err := strconv.Atoi(valkeyPort)
	if err != nil {
		return false, fmt.Errorf("invalid port: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", valkeyHost, port),
		Password: "testpass",
		DB:       0,
	})
	defer func() { _ = client.Close() }()

	_, err = client.Do(ctx, "ACL", "GETUSER", username).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateTestUserWithSelectorsInValkey creates a test user with selectors in Valkey for testing
func CreateTestUserWithSelectorsInValkey(ctx context.Context, username, password string) error {
	if valkeyHost == "" || valkeyPort == "" {
		return fmt.Errorf("valkey container not started")
	}

	port, err := strconv.Atoi(valkeyPort)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", valkeyHost, port),
		Password: "testpass",
		DB:       0,
	})
	defer func() { _ = client.Close() }()

	rules := []string{"reset", "on", ">" + password, "~*", "&*", "+@all", "(~key* +get)", "(~data* +set)"}
	return client.ACLSetUser(ctx, username, rules...).Err()
}

// ModifyUserInValkey modifies a user in Valkey with the given rules
func ModifyUserInValkey(ctx context.Context, username string, rules []string) error {
	if valkeyHost == "" || valkeyPort == "" {
		return fmt.Errorf("valkey container not started")
	}

	port, err := strconv.Atoi(valkeyPort)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", valkeyHost, port),
		Password: "testpass",
		DB:       0,
	})
	defer func() { _ = client.Close() }()

	return client.ACLSetUser(ctx, username, rules...).Err()
}
