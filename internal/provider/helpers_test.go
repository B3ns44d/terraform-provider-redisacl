// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestParseACLUser(t *testing.T) {
	tests := []struct {
		name     string
		acl      []interface{}
		expected *ACLUserResourceModel
	}{
		{
			name: "simple user",
			acl: []interface{}{
				"flags", []interface{}{"on"},
				"passwords", []interface{}{},
				"keys", "~*",
				"channels", "&*",
				"commands", "+@all",
			},
			expected: &ACLUserResourceModel{
				Enabled:  types.BoolValue(true),
				Keys:     types.StringValue("~*"),
				Channels: types.StringValue("&*"),
				Commands: types.StringValue("+@all"),
			},
		},
		{
			name: "disabled user",
			acl: []interface{}{
				"flags", []interface{}{"off"},
				"passwords", []interface{}{},
				"keys", "~somekey",
				"channels", "&somechannel",
				"commands", "-@all",
			},
			expected: &ACLUserResourceModel{
				Enabled:  types.BoolValue(false),
				Keys:     types.StringValue("~somekey"),
				Channels: types.StringValue("&somechannel"),
				Commands: types.StringValue("-@all"),
			},
		},
		{
			name: "user with selectors",
			acl: []interface{}{
				"flags", []interface{}{"on"},
				"passwords", []interface{}{},
				"keys", "~*",
				"channels", "&*",
				"commands", "+@all",
				"selectors", []interface{}{
					[]interface{}{"commands", "somecommand"},
				},
			},
			expected: &ACLUserResourceModel{
				Enabled:   types.BoolValue(true),
				Keys:      types.StringValue("~*"),
				Channels:  types.StringValue("&*"),
				Commands:  types.StringValue("+@all"),
				Selectors: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("somecommand")}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diags diag.Diagnostics
			actual := &ACLUserResourceModel{}
			parseACLUser(tt.acl, actual, &diags)

			assert.Empty(t, diags)
			assert.Equal(t, tt.expected.Enabled, actual.Enabled)
			assert.Equal(t, tt.expected.Keys, actual.Keys)
			assert.Equal(t, tt.expected.Channels, actual.Channels)
			assert.Equal(t, tt.expected.Commands, actual.Commands)
			if tt.expected.Selectors.IsNull() {
				assert.True(t, actual.Selectors.IsNull())
			} else {
				assert.True(t, tt.expected.Selectors.Equal(actual.Selectors))
			}
		})
	}
}

func TestBuildACLSetUserRules(t *testing.T) {
	tests := []struct {
		name     string
		data     *ACLUserResourceModel
		expected []string
	}{
		{
			name: "basic enabled user",
			data: &ACLUserResourceModel{
				Enabled:  types.BoolValue(true),
				Keys:     types.StringValue("~*"),
				Channels: types.StringValue("&*"),
				Commands: types.StringValue("+@all"),
			},
			expected: []string{"reset", "on", "resetkeys", "~*", "resetchannels", "&*", "-@all", "+@all"},
		},
		{
			name: "disabled user",
			data: &ACLUserResourceModel{
				Enabled:  types.BoolValue(false),
				Keys:     types.StringValue("~key*"),
				Channels: types.StringValue("&channel*"),
				Commands: types.StringValue("-@all +get"),
			},
			expected: []string{"reset", "off", "resetkeys", "~key*", "resetchannels", "&channel*", "-@all", "+get"},
		},
		{
			name: "commands without -@all prefix should get it added",
			data: &ACLUserResourceModel{
				Enabled:  types.BoolValue(true),
				Commands: types.StringValue("+get +set"),
			},
			expected: []string{"reset", "on", "~*", "&*", "-@all", "+get", "+set"},
		},
		{
			name: "commands with -@all prefix should not get duplicate",
			data: &ACLUserResourceModel{
				Enabled:  types.BoolValue(true),
				Commands: types.StringValue("-@all +get +set"),
			},
			expected: []string{"reset", "on", "~*", "&*", "-@all", "+get", "+set"},
		},
		{
			name: "user with single password",
			data: &ACLUserResourceModel{
				Enabled:   types.BoolValue(true),
				Passwords: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("password123")}),
			},
			expected: []string{"reset", "on", "resetpass", ">password123", "~*", "&*", "+@all"},
		},
		{
			name: "user with multiple passwords",
			data: &ACLUserResourceModel{
				Enabled: types.BoolValue(true),
				Passwords: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("password1"),
					types.StringValue("password2"),
					types.StringValue("password3"),
				}),
			},
			expected: []string{"reset", "on", "resetpass", ">password1", ">password2", ">password3", "~*", "&*", "+@all"},
		},
		{
			name: "nopass user (empty password list)",
			data: &ACLUserResourceModel{
				Enabled:   types.BoolValue(true),
				Passwords: types.ListValueMust(types.StringType, []attr.Value{}),
			},
			expected: []string{"reset", "on", "resetpass", "nopass", "~*", "&*", "+@all"},
		},
		{
			name: "user with multiple key patterns",
			data: &ACLUserResourceModel{
				Enabled: types.BoolValue(true),
				Keys:    types.StringValue("~key1* ~key2* ~key3*"),
			},
			expected: []string{"reset", "on", "resetkeys", "~key1*", "~key2*", "~key3*", "&*", "+@all"},
		},
		{
			name: "user with multiple channel patterns",
			data: &ACLUserResourceModel{
				Enabled:  types.BoolValue(true),
				Channels: types.StringValue("&channel1* &channel2*"),
			},
			expected: []string{"reset", "on", "~*", "resetchannels", "&channel1*", "&channel2*", "+@all"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := buildACLSetUserRules(tt.data)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestBuildACLSetUserRules_Comprehensive(t *testing.T) {
	tests := []struct {
		name     string
		data     *ACLUserResourceModel
		expected []string
	}{
		{
			name: "all parameters null",
			data: &ACLUserResourceModel{
				Enabled:  types.BoolNull(),
				Keys:     types.StringNull(),
				Channels: types.StringNull(),
				Commands: types.StringNull(),
			},
			expected: []string{"reset", "on", "~*", "&*", "+@all"},
		},
		{
			name: "with selectors",
			data: &ACLUserResourceModel{
				Enabled: types.BoolValue(true),
				Selectors: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("~key1* +get"),
					types.StringValue("~key2* +set"),
				}),
			},
			expected: []string{"reset", "on", "~*", "&*", "+@all", "(~key1* +get)", "(~key2* +set)"},
		},
		{
			name: "complex command combinations",
			data: &ACLUserResourceModel{
				Enabled:  types.BoolValue(true),
				Commands: types.StringValue("-@all +@read +@write -del"),
			},
			expected: []string{"reset", "on", "~*", "&*", "-@all", "+@read", "+@write", "-del"},
		},
		{
			name: "empty strings for keys and channels",
			data: &ACLUserResourceModel{
				Enabled:  types.BoolValue(true),
				Keys:     types.StringValue(""),
				Channels: types.StringValue(""),
				Commands: types.StringValue(""),
			},
			expected: []string{"reset", "on", "resetkeys", "resetchannels", "-@all"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := buildACLSetUserRules(tt.data)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
