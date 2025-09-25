// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func buildACLSetUserArgs(data *ACLUserResourceModel) []interface{} {
	args := []interface{}{"SETUSER", data.Name.ValueString()}

	if !data.Enabled.IsNull() {
		if data.Enabled.ValueBool() {
			args = append(args, "on")
		} else {
			args = append(args, "off")
		}
	}

	if !data.Passwords.IsNull() {
		for _, password := range data.Passwords.Elements() {
			args = append(args, ">"+password.(types.String).ValueString())
		}
	}

	if !data.Keys.IsNull() {
		args = append(args, "allkeys") // Reset keys
		args = append(args, data.Keys.ValueString())
	}

	if !data.Channels.IsNull() {
		args = append(args, "allchannels") // Reset channels
		args = append(args, data.Channels.ValueString())
	}

	if !data.Commands.IsNull() {
		args = append(args, "allcommands") // Reset commands
		args = append(args, data.Commands.ValueString())
	}

	if !data.Selectors.IsNull() {
		args = append(args, "clear-selectors") // Reset selectors
		for _, selector := range data.Selectors.Elements() {
			args = append(args, "("+selector.(types.String).ValueString()+")")
		}
	}

	return args
}

func parseACLUser(acl string, data *ACLUserResourceModel, diags *diag.Diagnostics) {
	parts := strings.Fields(acl)

	// The first part is always the user name
	data.Name = types.StringValue(parts[0])
	parts = parts[1:]

	var commands []string
	data.Enabled = types.BoolValue(false)
	for _, part := range parts {
		if part == "on" {
			data.Enabled = types.BoolValue(true)
		} else if part == "off" {
			data.Enabled = types.BoolValue(false)
		} else if strings.HasPrefix(part, "~") {
			data.Keys = types.StringValue(part)
		} else if strings.HasPrefix(part, "&") {
			data.Channels = types.StringValue(part)
		} else if strings.HasPrefix(part, ">") {
			// Passwords are not readable from Redis
		} else if strings.HasPrefix(part, "(") && strings.HasSuffix(part, ")") {
			selector := strings.TrimPrefix(strings.TrimSuffix(part, ")"), "(")
			selectors, d := types.ListValueFrom(context.Background(), types.StringType, []string{selector})
			if d.HasError() {
				diags.Append(d...)
				return
			}
			data.Selectors = selectors
		} else if part != "allkeys" && part != "allchannels" && part != "allcommands" && part != "nopass" {
			commands = append(commands, part)
		}
	}
	data.Commands = types.StringValue(strings.Join(commands, " "))
}