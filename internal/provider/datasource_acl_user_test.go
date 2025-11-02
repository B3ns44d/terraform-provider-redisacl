// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccACLUserDataSource_Read(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserDataSourceConfig_read("datasource_test_user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.redisacl_user.test", "name", "datasource_test_user"),
					resource.TestCheckResourceAttr("data.redisacl_user.test", "enabled", "true"),
					resource.TestCheckResourceAttr("data.redisacl_user.test", "keys", "~key*"),
					resource.TestCheckResourceAttr("data.redisacl_user.test", "channels", "&channel*"),
					resource.TestCheckResourceAttr("data.redisacl_user.test", "commands", "-@all +get +set"),
					// Verify resource and datasource have matching attributes
					resource.TestCheckResourceAttrPair("redisacl_user.source", "name", "data.redisacl_user.test", "name"),
					resource.TestCheckResourceAttrPair("redisacl_user.source", "enabled", "data.redisacl_user.test", "enabled"),
					resource.TestCheckResourceAttrPair("redisacl_user.source", "keys", "data.redisacl_user.test", "keys"),
					resource.TestCheckResourceAttrPair("redisacl_user.source", "channels", "data.redisacl_user.test", "channels"),
					resource.TestCheckResourceAttrPair("redisacl_user.source", "commands", "data.redisacl_user.test", "commands"),
				),
			},
		},
	})
}

func TestAccACLUserDataSource_NotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccACLUserDataSourceConfig_notFound("nonexistent_user"),
				ExpectError: regexp.MustCompile("ACL user nonexistent_user not found"),
			},
		},
	})
}

func TestAccACLUserDataSource_WithSelectors(t *testing.T) {
	t.Skip("Selectors may not be supported in Redis 7 Alpine or require different format")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			// Create user with selectors directly in Redis for testing
			ctx := context.Background()
			err := CreateTestUserWithSelectors(ctx, "selector_test_user", "testpass")
			if err != nil {
				t.Fatalf("Failed to create test user with selectors: %v", err)
			}
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserDataSourceConfig_withSelectors("selector_test_user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.redisacl_user.test", "name", "selector_test_user"),
					resource.TestCheckResourceAttr("data.redisacl_user.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("data.redisacl_user.test", "selectors.#"),
				),
			},
		},
	})
}

// Config helper functions

func testAccACLUserDataSourceConfig_read(name string) string {
	return fmt.Sprintf(`
provider "redisacl" {}

resource "redisacl_user" "source" {
  name     = "%s"
  enabled  = true
  keys     = "~key*"
  channels = "&channel*"
  commands = "-@all +get +set"
}

data "redisacl_user" "test" {
  name = redisacl_user.source.name
}
`, name)
}

func testAccACLUserDataSourceConfig_notFound(name string) string {
	return fmt.Sprintf(`
provider "redisacl" {}

data "redisacl_user" "test" {
  name = "%s"
}
`, name)
}

func testAccACLUserDataSourceConfig_withSelectors(name string) string {
	return fmt.Sprintf(`
provider "redisacl" {}

data "redisacl_user" "test" {
  name = "%s"
}
`, name)
}
