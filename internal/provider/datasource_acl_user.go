// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/redis/go-redis/v9"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ACLUserDataSource{}

func NewACLUserDataSource() datasource.DataSource {
	return &ACLUserDataSource{}
}

// ACLUserDataSource defines the data source implementation.
type ACLUserDataSource struct {
	client redis.UniversalClient
}

// ACLUserDataSourceModel describes the data source data model.
type ACLUserDataSourceModel struct {
	Name     types.String `tfsdk:"name"`
	Enabled  types.Bool   `tfsdk:"enabled"`
	Keys     types.String `tfsdk:"keys"`
	Channels types.String `tfsdk:"channels"`
	Commands types.String `tfsdk:"commands"`
	Selectors types.List   `tfsdk:"selectors"`
}

func (d *ACLUserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *ACLUserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets information about a Redis ACL user.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the user.",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the user is enabled.",
				Computed:            true,
			},
			"keys": schema.StringAttribute{
				MarkdownDescription: "The key patterns the user has access to.",
				Computed:            true,
			},
			"channels": schema.StringAttribute{
				MarkdownDescription: "The channel patterns the user has access to.",
				Computed:            true,
			},
			"commands": schema.StringAttribute{
				MarkdownDescription: "The commands the user can execute.",
				Computed:            true,
			},
			"selectors": schema.ListAttribute{
				MarkdownDescription: "A list of selectors for the user.",
				ElementType:         types.StringType,
				Computed:            true,
			},
		},
	}
}

func (d *ACLUserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(redis.UniversalClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected redis.UniversalClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *ACLUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ACLUserDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	val, err := d.client.Do(ctx, "ACL", "GETUSER", data.Name.ValueString()).Result()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read ACL user, got error: %s", err))
		return
	}

	if val == nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("ACL user %s not found", data.Name.ValueString()))
		return
	}

	aclUserResourceModel := &ACLUserResourceModel{}
	parseACLUser(val.(string), aclUserResourceModel, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Enabled = aclUserResourceModel.Enabled
	data.Keys = aclUserResourceModel.Keys
	data.Channels = aclUserResourceModel.Channels
	data.Commands = aclUserResourceModel.Commands
	data.Selectors = aclUserResourceModel.Selectors

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}