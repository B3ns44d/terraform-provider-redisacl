// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// This is the struct we'll pass to datasources and resources.
type RedisClient struct {
	client UniversalClient
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
	UseValkey             types.Bool   `tfsdk:"use_valkey"`
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

func (p *RedisACLProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "redisacl"
	resp.Version = p.version
}

func (p *RedisACLProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
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
			"use_valkey": schema.BoolAttribute{
				MarkdownDescription: "Whether to use Valkey instead of Redis as the backend. When set to `true`, the provider will use the Valkey Glide client. When set to `false` or omitted, the provider will use the Redis client (default behavior).",
				Optional:            true,
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

	// Build TLS configuration if enabled
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

	// Determine which client to create based on use_valkey flag
	var client UniversalClient
	var err error
	useValkey := data.UseValkey.ValueBool()

	if useValkey {
		// Create Valkey client
		client, err = createValkeyClient(ctx, data, tlsConfig)
		if err != nil {
			resp.Diagnostics.AddError("Valkey Client Configuration",
				fmt.Sprintf("Unable to create Valkey client: %s", err))
			return
		}
	} else {
		// Create Redis client (default behavior)
		client, err = createRedisClient(ctx, data, tlsConfig)
		if err != nil {
			resp.Diagnostics.AddError("Redis Client Configuration",
				fmt.Sprintf("Unable to create Redis client: %s", err))
			return
		}
	}

	// Test connection with backend-specific error message
	_, err = client.Ping(ctx)
	if err != nil {
		backendType := "Redis"
		if useValkey {
			backendType = "Valkey"
		}
		resp.Diagnostics.AddError("Client Configuration",
			fmt.Sprintf("Unable to connect to %s: %s", backendType, err))
		return
	}

	// Create RedisClient wrapper with the universal client interface
	redisClient := &RedisClient{
		client: client,
		mutex:  &sync.Mutex{},
	}

	resp.DataSourceData = redisClient
	resp.ResourceData = redisClient
}

func (p *RedisACLProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewACLUserResource,
	}
}

func (p *RedisACLProvider) DataSources(_ context.Context) []func() datasource.DataSource {
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
