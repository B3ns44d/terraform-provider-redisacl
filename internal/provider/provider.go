// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/redis/go-redis/v9"
)

// This is the struct we'll pass to datasources and resources.
type RedisClient struct {
	client redis.UniversalClient
	mutex  *sync.Mutex
}

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
	Address               types.String `tfsdk:"address"`
	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password"`
	UseTLS                types.Bool   `tfsdk:"use_tls"`
	TLSCACert             types.String `tfsdk:"tls_ca_cert"`
	TLSCert               types.String `tfsdk:"tls_cert"`
	TLSKey                types.String `tfsdk:"tls_key"`
	TLSInsecureSkipVerify types.Bool   `tfsdk:"tls_insecure_skip_verify"`
	Sentinel              types.Object `tfsdk:"sentinel"`
	Cluster               types.Object `tfsdk:"cluster"`
}

type SentinelModel struct {
	MasterName types.String   `tfsdk:"master_name"`
	Addresses  []types.String `tfsdk:"addresses"`
	Username   types.String   `tfsdk:"username"`
	Password   types.String   `tfsdk:"password"`
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
			"tls_ca_cert": schema.StringAttribute{
				MarkdownDescription: "PEM-encoded CA certificate for TLS verification.",
				Optional:            true,
				Sensitive:           true,
			},
			"tls_cert": schema.StringAttribute{
				MarkdownDescription: "PEM-encoded client certificate for mutual TLS.",
				Optional:            true,
				Sensitive:           true,
			},
			"tls_key": schema.StringAttribute{
				MarkdownDescription: "PEM-encoded client private key for mutual TLS.",
				Optional:            true,
				Sensitive:           true,
			},
			"tls_insecure_skip_verify": schema.BoolAttribute{
				MarkdownDescription: "Disable TLS certificate verification (insecure, use only for testing).",
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
	var tlsConfig *tls.Config
	if data.UseTLS.ValueBool() {
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12, // Set minimum TLS version for security
		}
		if data.TLSInsecureSkipVerify.ValueBool() {
			tlsConfig.InsecureSkipVerify = true
		}
		if !data.TLSCACert.IsNull() {
			caCertPool := x509.NewCertPool()
			if ok := caCertPool.AppendCertsFromPEM([]byte(data.TLSCACert.ValueString())); !ok {
				resp.Diagnostics.AddError("TLS Configuration", "Failed to parse CA certificate")
				return
			}
			tlsConfig.RootCAs = caCertPool
		}
		if !data.TLSCert.IsNull() && !data.TLSKey.IsNull() {
			cert, err := tls.X509KeyPair([]byte(data.TLSCert.ValueString()), []byte(data.TLSKey.ValueString()))
			if err != nil {
				resp.Diagnostics.AddError("TLS Configuration", fmt.Sprintf("Failed to load client cert/key: %s", err))
				return
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
	}
	var client redis.UniversalClient
	// Override with REDIS_URL environment variable if set
	redisURL := os.Getenv("REDIS_URL")
	if redisURL != "" {
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			resp.Diagnostics.AddError("Client Configuration", fmt.Sprintf("Invalid Redis URL: %s", err))
			return
		}
		if tlsConfig != nil {
			opts.TLSConfig = tlsConfig
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
		resp.Diagnostics.Append(data.Cluster.As(ctx, &clusterModel, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
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
	// Check the connection
	_, err := client.Ping(ctx).Result()
	if err != nil {
		resp.Diagnostics.AddError("Client Configuration", fmt.Sprintf("Unable to connect to Redis: %s", err))
		return
	}
	redisClient := &RedisClient{
		client: client,
		mutex:  &sync.Mutex{},
	}
	resp.DataSourceData = redisClient
	resp.ResourceData = redisClient
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
