// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/redis/go-redis/v9"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ACLUserResource{}
var _ resource.ResourceWithImportState = &ACLUserResource{}

func NewACLUserResource() resource.Resource {
	return &ACLUserResource{}
}

// ACLUserResource defines the resource implementation.
type ACLUserResource struct {
	redisClient *RedisClient
}

// ACLUserResourceModel describes the resource data model.
type ACLUserResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	Passwords         types.List   `tfsdk:"passwords"`
	Keys              types.String `tfsdk:"keys"`
	Channels          types.String `tfsdk:"channels"`
	Commands          types.String `tfsdk:"commands"`
	Selectors         types.List   `tfsdk:"selectors"`
	AllowSelfMutation types.Bool   `tfsdk:"allow_self_mutation"`
}

func (r *ACLUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *ACLUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Redis ACL user.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the user (same as name).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the user.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the user is enabled.",
				Optional:            true,
			},
			"passwords": schema.ListAttribute{
				MarkdownDescription: "A list of passwords for the user.",
				ElementType:         types.StringType,
				Optional:            true,
				Sensitive:           true,
			},
			"keys": schema.StringAttribute{
				MarkdownDescription: "The key patterns the user has access to (space-separated if multiple).",
				Optional:            true,
			},
			"channels": schema.StringAttribute{
				MarkdownDescription: "The channel patterns the user has access to (space-separated if multiple).",
				Optional:            true,
			},
			"commands": schema.StringAttribute{
				MarkdownDescription: "The commands the user can execute (space-separated).",
				Optional:            true,
			},
			"selectors": schema.ListAttribute{
				MarkdownDescription: "A list of selectors for the user (each a string of space-separated rules).",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"allow_self_mutation": schema.BoolAttribute{
				MarkdownDescription: "Whether to allow the user to modify itself.",
				Optional:            true,
			},
		},
	}
}

func (r *ACLUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	redisClient, ok := req.ProviderData.(*RedisClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *RedisClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.redisClient = redisClient
}

func (r *ACLUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ACLUserResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	rules := buildACLSetUserRules(&data)

	err := r.redisClient.client.ACLSetUser(ctx, data.Name.ValueString(), rules...).Err()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create ACL user, got error: %s", err))
		return
	}

	// Set the ID to the user name
	data.ID = data.Name

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ACLUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ACLUserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	r.redisClient.mutex.Lock()
	defer r.redisClient.mutex.Unlock()

	result, err := r.redisClient.client.Do(ctx, "ACL", "GETUSER", data.Name.ValueString()).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read ACL user result, got error: %s", err))
		return
	}

	var val []interface{}
	switch res := result.(type) {
	case []interface{}:
		val = res
	case map[interface{}]interface{}:
		for k, v := range res {
			val = append(val, k, v)
		}
	default:
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse ACL GETUSER response: unexpected type %T", result))
		return
	}

	if len(val) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	parseACLUser(val, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Ensure ID is set
	data.ID = data.Name

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ACLUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ACLUserResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check for self-mutation
	if !data.AllowSelfMutation.ValueBool() {
		result, err := r.redisClient.client.Do(ctx, "ACL", "WHOAMI").Result()
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get current user, got error: %s", err))
			return
		}
		currentUser, ok := result.(string)
		if !ok {
			resp.Diagnostics.AddError("Client Error", "Unable to parse current user response")
			return
		}
		if currentUser == data.Name.ValueString() {
			resp.Diagnostics.AddError("Self-Mutation Error", "Cannot modify the currently authenticated user without setting allow_self_mutation to true")
			return
		}
	}

	rules := buildACLSetUserRules(&data)

	err := r.redisClient.client.ACLSetUser(ctx, data.Name.ValueString(), rules...).Err()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update ACL user, got error: %s", err))
		return
	}

	// Ensure ID is set
	data.ID = data.Name

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ACLUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ACLUserResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check for self-mutation
	if !data.AllowSelfMutation.ValueBool() {
		result, err := r.redisClient.client.Do(ctx, "ACL", "WHOAMI").Result()
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get current user, got error: %s", err))
			return
		}
		currentUser, ok := result.(string)
		if !ok {
			resp.Diagnostics.AddError("Client Error", "Unable to parse current user response")
			return
		}
		if currentUser == data.Name.ValueString() {
			resp.Diagnostics.AddError("Self-Mutation Error", "Cannot delete the currently authenticated user without setting allow_self_mutation to true")
			return
		}
	}

	err := r.redisClient.client.ACLDelUser(ctx, data.Name.ValueString()).Err()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete ACL user, got error: %s", err))
		return
	}
}

func (r *ACLUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
