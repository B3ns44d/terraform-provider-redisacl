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
		acl      string
		expected *ACLUserResourceModel
	}{
		{
			name: "simple user",
			acl:  "myuser on ~* &* +@all",
			expected: &ACLUserResourceModel{
				Name:     types.StringValue("myuser"),
				Enabled:  types.BoolValue(true),
				Keys:     types.StringValue("~*"),
				Channels: types.StringValue("&*"),
				Commands: types.StringValue("+@all"),
			},
		},
		{
			name: "disabled user",
			acl:  "anotheruser off ~somekey &somechannel -@all",
			expected: &ACLUserResourceModel{
				Name:     types.StringValue("anotheruser"),
				Enabled:  types.BoolValue(false),
				Keys:     types.StringValue("~somekey"),
				Channels: types.StringValue("&somechannel"),
				Commands: types.StringValue("-@all"),
			},
		},
		{
			name: "user with selectors",
			acl:  "selectoruser on (somecommand) ~* &* +@all",
			expected: &ACLUserResourceModel{
				Name:      types.StringValue("selectoruser"),
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
			assert.Equal(t, tt.expected.Name, actual.Name)
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