// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/redis/go-redis/v9"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ACLUserResource{}
var _ resource.ResourceWithImportState = &ACLUserResource{}

func NewACLUserResource() resource.Resource {
	return &ACLUserResource{}
}

// ACLUserResource defines the resource implementation.
type ACLUserResource struct {
	client redis.UniversalClient
}

// ACLUserResourceModel describes the resource data model.
type ACLUserResourceModel struct {
	Name             types.String `tfsdk:"name"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	Passwords        types.List   `tfsdk:"passwords"`
	Keys             types.String `tfsdk:"keys"`
	Channels         types.String `tfsdk:"channels"`
	Commands         types.String `tfsdk:"commands"`
	Selectors        types.List   `tfsdk:"selectors"`
	AllowSelfMutation types.Bool `tfsdk:"allow_self_mutation"`
}

func (r *ACLUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *ACLUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Redis ACL user.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the user.",
				Required:            true,
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
				MarkdownDescription: "The key patterns the user has access to.",
				Optional:            true,
			},
			"channels": schema.StringAttribute{
				MarkdownDescription: "The channel patterns the user has access to.",
				Optional:            true,
			},
			"commands": schema.StringAttribute{
				MarkdownDescription: "The commands the user can execute.",
				Optional:            true,
			},
			"selectors": schema.ListAttribute{
				MarkdownDescription: "A list of selectors for the user.",
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

	client, ok := req.ProviderData.(redis.UniversalClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected redis.UniversalClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *ACLUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ACLUserResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	args := buildACLSetUserArgs(&data)

	err := r.client.Do(ctx, args...).Err()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create ACL user, got error: %s", err))
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ACLUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ACLUserResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	val, err := r.client.Do(ctx, "ACL", "GETUSER", data.Name.ValueString()).Result()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read ACL user, got error: %s", err))
		return
	}

	if val == nil {
		// User does not exist
		resp.State.RemoveResource(ctx)
		return
	}

	parseACLUser(val.(string), &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ACLUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ACLUserResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check for self-mutation
	if !data.AllowSelfMutation.ValueBool() {
		currentUser, err := r.client.Do(ctx, "ACL", "WHOAMI").Result()
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get current user, got error: %s", err))
			return
		}
		if currentUser.(string) == data.Name.ValueString() {
			resp.Diagnostics.AddError("Self-Mutation Error", "Cannot modify the currently authenticated user without setting allow_self_mutation to true")
			return
		}
	}

	args := buildACLSetUserArgs(&data)

	err := r.client.Do(ctx, args...).Err()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update ACL user, got error: %s", err))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ACLUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ACLUserResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check for self-mutation
	if !data.AllowSelfMutation.ValueBool() {
		currentUser, err := r.client.Do(ctx, "ACL", "WHOAMI").Result()
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get current user, got error: %s", err))
			return
		}
		if currentUser.(string) == data.Name.ValueString() {
			resp.Diagnostics.AddError("Self-Mutation Error", "Cannot delete the currently authenticated user without setting allow_self_mutation to true")
			return
		}
	}

	err := r.client.Do(ctx, "ACL", "DELUSER", data.Name.ValueString()).Err()
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete ACL user, got error: %s", err))
		return
	}
}

func (r *ACLUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}