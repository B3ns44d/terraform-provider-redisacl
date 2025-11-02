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
var _ datasource.DataSource = &ACLUsersDataSource{}

func NewACLUsersDataSource() datasource.DataSource {
	return &ACLUsersDataSource{}
}

// ACLUsersDataSource defines the data source implementation.
type ACLUsersDataSource struct {
	redisClient *RedisClient
}

// ACLUsersDataSourceModel describes the data source data model.
type ACLUsersDataSourceModel struct {
	Users []ACLUserDataSourceModel `tfsdk:"users"`
}

func (d *ACLUsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *ACLUsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets information about all Redis ACL users.",

		Attributes: map[string]schema.Attribute{
			"users": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the user.",
							Computed:            true,
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
				},
			},
		},
	}
}

func (d *ACLUsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	redisClient, ok := req.ProviderData.(*RedisClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *RedisClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.redisClient = redisClient
}

func (d *ACLUsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ACLUsersDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	d.redisClient.mutex.Lock()
	defer d.redisClient.mutex.Unlock()

	usernames, err := d.redisClient.client.Do(ctx, "ACL", "USERS").StringSlice()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list ACL users, got error: %s", err))
		return
	}

	data.Users = []ACLUserDataSourceModel{}
	for _, username := range usernames {
		result, err := d.redisClient.client.Do(ctx, "ACL", "GETUSER", username).Result()
		if err != nil {
			if err != redis.Nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get ACL user %s, got error: %s", username, err))
			}
			continue
		}

		var userVal []interface{}
		switch res := result.(type) {
		case []interface{}:
			userVal = res
		case map[interface{}]interface{}:
			for k, v := range res {
				userVal = append(userVal, k, v)
			}
		default:
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse ACL GETUSER response for user %s: unexpected type %T", username, result))
			continue
		}

		if len(userVal) == 0 {
			continue
		}

		var userModel ACLUserDataSourceModel
		userModel.Name = types.StringValue(username)
		temp := &ACLUserResourceModel{}
		parseACLUser(userVal, temp, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		userModel.Enabled = temp.Enabled
		userModel.Keys = temp.Keys
		userModel.Channels = temp.Channels
		userModel.Commands = temp.Commands
		userModel.Selectors = temp.Selectors

		data.Users = append(data.Users, userModel)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
