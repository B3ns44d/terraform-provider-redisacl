package provider

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisACLClient interface {
	Do(ctx context.Context, args ...interface{}) *redis.Cmd
	ACLSetUser(ctx context.Context, username string, rules ...string) *redis.StatusCmd
	ACLDelUser(ctx context.Context, username string) *redis.IntCmd
	Ping(ctx context.Context) *redis.StatusCmd
}

func NewFailoverCompositeClient(ctx context.Context, opts *redis.FailoverOptions) RedisACLClient {
	sentinelClient := redis.NewSentinelClient(&redis.Options{
		Addr: opts.SentinelAddrs[0], // taking first server just to pull failover structure
	})

	replicaAddrsCmdResult := sentinelClient.Replicas(ctx, opts.MasterName)
	replicaAddrs := extractReplicaAddrs(replicaAddrsCmdResult)

	client := &failoverCompositeClient{
		masterClient:   redis.NewFailoverClient(opts),
		replicaClients: make([]RedisACLClient, len(replicaAddrs)),
	}

	for i, replicaAddr := range replicaAddrs {
		client.replicaClients[i] = redis.NewClient(&redis.Options{
			Addr:      replicaAddr,
			Username:  opts.Username,
			Password:  opts.Password,
			TLSConfig: opts.TLSConfig,
		})
	}

	cmd := client.masterClient.Do(ctx, "ACL", "LOAD")
	if cmd.Err() == nil {
		client.aclSave = true
	}

	return client
}

type failoverCompositeClient struct {
	masterClient   RedisACLClient
	replicaClients []RedisACLClient
	aclSave        bool
}

func (c *failoverCompositeClient) Do(ctx context.Context, args ...interface{}) *redis.Cmd {
	return c.masterClient.Do(ctx, args...)
}

func (c *failoverCompositeClient) ACLSetUser(ctx context.Context, username string, rules ...string) *redis.StatusCmd {
	return mutateAndSaveAclOnAllServers(ctx, c, func(aclClient RedisACLClient) *redis.StatusCmd {
		return aclClient.ACLSetUser(ctx, username, rules...)
	})
}

func (c *failoverCompositeClient) ACLDelUser(ctx context.Context, username string) *redis.IntCmd {
	return mutateAndSaveAclOnAllServers(ctx, c, func(aclClient RedisACLClient) *redis.IntCmd {
		return aclClient.ACLDelUser(ctx, username)
	})
}

func (c *failoverCompositeClient) Ping(ctx context.Context) *redis.StatusCmd {
	result := c.masterClient.Ping(ctx)
	if result.Err() != nil {
		return result
	}

	for _, replicaClient := range c.replicaClients {
		result = replicaClient.Ping(ctx)
		if result.Err() != nil {
			return result
		}
	}

	// to conform with api - returning last successfull result of command
	return result
}

func mutateAndSaveAclOnAllServers[TResult redis.Cmder](ctx context.Context, client *failoverCompositeClient, operation func(RedisACLClient) TResult) TResult {
	clients := append(client.replicaClients, client.masterClient)
	var result TResult

	for _, c := range clients {
		result = operation(c)
		if result.Err() != nil {
			return result
		}

		if client.aclSave {
			err := aclSave(ctx, c)
			if err != nil {
				result.SetErr(err)
				return result
			}
		}
	}

	// to conform with api - returning last successfull result of command
	return result
}

func extractReplicaAddrs(redisRawResult *redis.MapStringStringSliceCmd) []string {
	result := make([]string, len(redisRawResult.Val()))

	for i, v := range redisRawResult.Val() {
		result[i] = fmt.Sprintf("%s:%s", v["ip"], v["port"])
	}

	return result
}

func aclSave(ctx context.Context, client RedisACLClient) error {
	result := client.Do(ctx, "ACL", "SAVE")
	if result.Err() != nil {
		return result.Err()
	}

	return nil
}
