// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func buildACLSetUserRules(data *ACLUserResourceModel) []string {
	rules := []string{"reset"}

	if data.Enabled.IsNull() {
		rules = append(rules, "on")
	} else {
		if data.Enabled.ValueBool() {
			rules = append(rules, "on")
		} else {
			rules = append(rules, "off")
		}
	}

	if !data.Passwords.IsNull() {
		rules = append(rules, "resetpass")
		if len(data.Passwords.Elements()) == 0 {
			rules = append(rules, "nopass")
		} else {
			for _, password := range data.Passwords.Elements() {
				rules = append(rules, ">"+password.(types.String).ValueString())
			}
		}
	}

	if !data.Keys.IsNull() {
		rules = append(rules, "resetkeys")
		rules = append(rules, strings.Fields(data.Keys.ValueString())...)
	} else {
		rules = append(rules, "~*")
	}

	if !data.Channels.IsNull() {
		rules = append(rules, "resetchannels")
		rules = append(rules, strings.Fields(data.Channels.ValueString())...)
	} else {
		rules = append(rules, "&*")
	}

	if !data.Commands.IsNull() {
		rules = append(rules, "-@all")
		rules = append(rules, strings.Fields(data.Commands.ValueString())...)
	} else {
		rules = append(rules, "+@all")
	}

	if !data.Selectors.IsNull() {
		for _, selector := range data.Selectors.Elements() {
			rules = append(rules, "("+selector.(types.String).ValueString()+")")
		}
	}

	return rules
}

func parseACLUser(acl []interface{}, data *ACLUserResourceModel, diags *diag.Diagnostics) {
	data.Enabled = types.BoolValue(false)

	for i := 0; i < len(acl); i += 2 {
		key := acl[i].(string)
		v := acl[i+1]

		switch key {
		case "flags":
			flags, ok := v.([]interface{})
			if !ok {
				diags.AddError("Parse Error", "flags not array")
				return
			}
			for _, f := range flags {
				if f.(string) == "on" {
					data.Enabled = types.BoolValue(true)
				}
			}
		case "passwords":
			// Skip passwords
		case "keys":
			var keyStr string
			switch vv := v.(type) {
			case string:
				keyStr = vv
			case []interface{}:
				var parts []string
				for _, p := range vv {
					parts = append(parts, p.(string))
				}
				keyStr = strings.Join(parts, " ")
			default:
				diags.AddError("Parse Error", "keys not string or array")
				return
			}
			data.Keys = types.StringValue(keyStr)
		case "channels":
			var chanStr string
			switch vv := v.(type) {
			case string:
				chanStr = vv
			case []interface{}:
				var parts []string
				for _, p := range vv {
					parts = append(parts, p.(string))
				}
				chanStr = strings.Join(parts, " ")
			default:
				diags.AddError("Parse Error", "channels not string or array")
				return
			}
			data.Channels = types.StringValue(chanStr)
		case "commands":
			cmdStr, ok := v.(string)
			if !ok {
				diags.AddError("Parse Error", "commands not string")
				return
			}
			data.Commands = types.StringValue(cmdStr)
		case "selectors":
			sels, ok := v.([]interface{})
			if !ok {
				diags.AddError("Parse Error", "selectors not array")
				return
			}
			var selectorStrs []string
			for _, selI := range sels {
				sel, ok := selI.([]interface{})
				if !ok {
					diags.AddError("Parse Error", "selector not array")
					return
				}
				var parts []string
				for j := 0; j < len(sel); j += 2 {
					sk := sel[j].(string)
					svI := sel[j+1]
					var sv string
					switch svv := svI.(type) {
					case string:
						sv = svv
					case []interface{}:
						var pp []string
						for _, ppp := range svv {
							pp = append(pp, ppp.(string))
						}
						sv = strings.Join(pp, " ")
					default:
						diags.AddError("Parse Error", "selector value not string or array")
						return
					}
					if sk == "commands" || sk == "keys" || sk == "channels" {
						parts = append(parts, sv)
					}
				}
				selectorStrs = append(selectorStrs, strings.Join(parts, " "))
			}
			selectors, d := types.ListValueFrom(context.Background(), types.StringType, selectorStrs)
			diags.Append(d...)
			data.Selectors = selectors
		}
	}
}
