// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/redis/go-redis/v9"
)

// Ensure RedisACLProvider satisfies various provider interfaces.
var _ provider.Provider = &RedisACLProvider{}

// RedisACLProvider defines the provider implementation.
type RedisACLProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// RedisACLProviderModel describes the provider data model.
type RedisACLProviderModel struct {
	Address    types.String `tfsdk:"address"`
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
	UseTLS     types.Bool   `tfsdk:"use_tls"`
	Sentinel   types.Object `tfsdk:"sentinel"`
	Cluster    types.Object `tfsdk:"cluster"`
}

type SentinelModel struct {
	MasterName    types.String   `tfsdk:"master_name"`
	Addresses     []types.String `tfsdk:"addresses"`
	Username      types.String   `tfsdk:"username"`
	Password      types.String   `tfsdk:"password"`
}

type ClusterModel struct {
	Addresses []types.String `tfsdk:"addresses"`
	Username  types.String   `tfsdk:"username"`
	Password  types.String   `tfsdk:"password"`
}

func (p *RedisACLProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "redisacl"
	resp.Version = p.version
}

func (p *RedisACLProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"address": schema.StringAttribute{
				MarkdownDescription: "The address of the Redis server.",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username for Redis authentication.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password for Redis authentication.",
				Optional:            true,
				Sensitive:           true,
			},
			"use_tls": schema.BoolAttribute{
				MarkdownDescription: "Whether to use TLS for the connection.",
				Optional:            true,
			},
			"sentinel": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for Redis Sentinel.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"master_name": schema.StringAttribute{
						MarkdownDescription: "The name of the Sentinel master.",
						Required:            true,
					},
					"addresses": schema.ListAttribute{
						ElementType:         types.StringType,
						MarkdownDescription: "A list of Sentinel addresses.",
						Required:            true,
					},
					"username": schema.StringAttribute{
						MarkdownDescription: "The username for Sentinel authentication.",
						Optional:            true,
					},
					"password": schema.StringAttribute{
						MarkdownDescription: "The password for Sentinel authentication.",
						Optional:            true,
						Sensitive:           true,
					},
				},
			},
			"cluster": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for Redis Cluster.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"addresses": schema.ListAttribute{
						ElementType:         types.StringType,
						MarkdownDescription: "A list of cluster node addresses.",
						Required:            true,
					},
					"username": schema.StringAttribute{
						MarkdownDescription: "The username for cluster authentication.",
						Optional:            true,
					},
					"password": schema.StringAttribute{
						MarkdownDescription: "The password for cluster authentication.",
						Optional:            true,
						Sensitive:           true,
					},
				},
			},
		},
	}
}

func (p *RedisACLProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data RedisACLProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var client redis.UniversalClient

	// You can override the endpoint address with the REDIS_URL environment variable.
	redisURL := os.Getenv("REDIS_URL")
	if redisURL != "" {
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			resp.Diagnostics.AddError("Client Configuration", fmt.Sprintf("Invalid Redis URL: %s", err))
			return
		}
		client = redis.NewClient(opts)
	} else if !data.Sentinel.IsNull() {
		// Sentinel configuration
		var sentinelModel SentinelModel
		resp.Diagnostics.Append(data.Sentinel.As(ctx, &sentinelModel, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		var sentinelAddrs []string
		for _, addr := range sentinelModel.Addresses {
			sentinelAddrs = append(sentinelAddrs, addr.ValueString())
		}

		opts := &redis.FailoverOptions{
			MasterName:       sentinelModel.MasterName.ValueString(),
			SentinelAddrs:    sentinelAddrs,
			Username:         sentinelModel.Username.ValueString(),
			Password:         sentinelModel.Password.ValueString(),
		}
		if data.UseTLS.ValueBool() {
			opts.TLSConfig = &tls.Config{}
		}
		client = redis.NewFailoverClient(opts)

	} else if !data.Cluster.IsNull() {
		// Cluster configuration
		var clusterModel ClusterModel
		resp.Diagnostics.Append(data.Cluster.As(ctx, &clusterModel, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		var clusterAddrs []string
		for _, addr := range clusterModel.Addresses {
			clusterAddrs = append(clusterAddrs, addr.ValueString())
		}

		opts := &redis.ClusterOptions{
			Addrs:    clusterAddrs,
			Username: clusterModel.Username.ValueString(),
			Password: clusterModel.Password.ValueString(),
		}
		if data.UseTLS.ValueBool() {
			opts.TLSConfig = &tls.Config{}
		}
		client = redis.NewClusterClient(opts)
	} else {
		// Single instance configuration
		address := "localhost:6379"
		if !data.Address.IsNull() {
			address = data.Address.ValueString()
		}
		opts := &redis.Options{
			Addr:     address,
			Username: data.Username.ValueString(),
			Password: data.Password.ValueString(),
		}
		if data.UseTLS.ValueBool() {
			opts.TLSConfig = &tls.Config{}
		}
		client = redis.NewClient(opts)
	}

	// Check the connection
	_, err := client.Ping(ctx).Result()
	if err != nil {
		resp.Diagnostics.AddError("Client Configuration", fmt.Sprintf("Unable to connect to Redis: %s", err))
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *RedisACLProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewACLUserResource,
	}
}

func (p *RedisACLProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewACLUserDataSource,
		NewACLUsersDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RedisACLProvider{
			version: version,
		}
	}
}