// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccACLUserResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccACLUserResourceConfig("testuser", true, "testpassword", "~*", "&*", "+@all"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "testuser"),
					resource.TestCheckResourceAttr("redisacl_user.test", "enabled", "true"),
					resource.TestCheckResourceAttr("redisacl_user.test", "keys", "~*"),
					resource.TestCheckResourceAttr("redisacl_user.test", "channels", "&*"),
					resource.TestCheckResourceAttr("redisacl_user.test", "commands", "+@all"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "redisacl_user.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccACLUserResourceConfig("testuser", false, "newpassword", "~other", "&other", "-@all"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "testuser"),
					resource.TestCheckResourceAttr("redisacl_user.test", "enabled", "false"),
					resource.TestCheckResourceAttr("redisacl_user.test", "keys", "~other"),
					resource.TestCheckResourceAttr("redisacl_user.test", "channels", "&other"),
					resource.TestCheckResourceAttr("redisacl_user.test", "commands", "-@all"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccACLUserResourceConfig(name string, enabled bool, password, keys, channels, commands string) string {
	return fmt.Sprintf(`
provider "redisacl" {
  // Please ensure REDIS_URL is set in your environment
}

resource "redisacl_user" "test" {
  name     = "%s"
  enabled  = %t
  passwords = ["%s"]
  keys     = "%s"
  channels = "%s"
  commands = "%s"
}
`, name, enabled, password, keys, channels, commands)
}