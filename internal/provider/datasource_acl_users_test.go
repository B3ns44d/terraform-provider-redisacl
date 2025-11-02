// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccACLUsersDataSource_ReadAll(t *testing.T) {
	testUsers := []string{"multi_user_1", "multi_user_2", "multi_user_3"}
	
	resource.Test(t, resource.TestCase{
		PreCheck: func() { 
			testAccPreCheck(t)
			// Create multiple test users after cleanup
			ctx := context.Background()
			for _, user := range testUsers {
				err := CreateTestUser(ctx, user, "testpass")
				if err != nil {
					t.Fatalf("Failed to create test user %s: %v", user, err)
				}
			}
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUsersDataSourceConfig_readAll(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check that we have at least the test users we created (plus default user)
					resource.TestCheckResourceAttrWith("data.redisacl_users.test", "users.#", func(value string) error {
						if value == "0" {
							return fmt.Errorf("Expected at least 1 user, got 0")
						}
						return nil
					}),
					// Verify that our test users are in the list
					testAccCheckUsersContain("data.redisacl_users.test", testUsers),
				),
			},
		},
	})
}

func TestAccACLUsersDataSource_WithResources(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUsersDataSourceConfig_withResources(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check that we have at least the users we created via resources
					resource.TestCheckResourceAttrWith("data.redisacl_users.test", "users.#", func(value string) error {
						if value == "0" {
							return fmt.Errorf("Expected at least 1 user, got 0")
						}
						return nil
					}),
					// Verify specific users exist in the datasource
					testAccCheckUsersContain("data.redisacl_users.test", []string{"resource_user_1", "resource_user_2"}),
				),
			},
		},
	})
}

func TestAccACLUsersDataSource_Empty(t *testing.T) {
	ctx := context.Background()
	
	// Clean up all test users to ensure clean slate
	CleanupRedisUsers(ctx)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUsersDataSourceConfig_empty(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Should have at least the default user
					resource.TestCheckResourceAttrWith("data.redisacl_users.test", "users.#", func(value string) error {
						if value == "0" {
							return fmt.Errorf("Expected at least the default user, got 0")
						}
						return nil
					}),
					// Verify default user exists
					testAccCheckUsersContain("data.redisacl_users.test", []string{"default"}),
				),
			},
		},
	})
}

func TestAccACLUsersDataSource_UserAttributes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { 
			testAccPreCheck(t)
			// Create a specific test user after cleanup
			ctx := context.Background()
			err := CreateTestUser(ctx, "attr_test_user", "testpass")
			if err != nil {
				t.Fatalf("Failed to create test user: %v", err)
			}
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUsersDataSourceConfig_userAttributes(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith("data.redisacl_users.test", "users.#", func(value string) error {
						if value == "0" {
							return fmt.Errorf("Expected at least 1 user, got 0")
						}
						return nil
					}),
					// Check that user attributes are properly populated
					testAccCheckUserAttributesPopulated("data.redisacl_users.test"),
				),
			},
		},
	})
}

// Helper functions

func testAccCheckUsersContain(resourceName string, expectedUsers []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		usersCount := rs.Primary.Attributes["users.#"]
		if usersCount == "0" {
			return fmt.Errorf("No users found in datasource")
		}

		// Check if each expected user exists in the datasource
		for _, expectedUser := range expectedUsers {
			found := false
			for i := 0; i < 10; i++ { // Check up to 10 users (should be enough for tests)
				userNameKey := fmt.Sprintf("users.%d.name", i)
				if userName, exists := rs.Primary.Attributes[userNameKey]; exists && userName == expectedUser {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("Expected user %s not found in datasource", expectedUser)
			}
		}

		return nil
	}
}

func testAccCheckUserAttributesPopulated(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		// Check that the first user has all required attributes
		requiredAttrs := []string{"users.0.name", "users.0.enabled", "users.0.keys", "users.0.channels", "users.0.commands"}
		for _, attr := range requiredAttrs {
			if _, exists := rs.Primary.Attributes[attr]; !exists {
				return fmt.Errorf("Required attribute %s not found", attr)
			}
		}

		return nil
	}
}

// Config helper functions

func testAccACLUsersDataSourceConfig_readAll() string {
	return `
provider "redisacl" {}

data "redisacl_users" "test" {}
`
}

func testAccACLUsersDataSourceConfig_withResources() string {
	return `
provider "redisacl" {}

resource "redisacl_user" "test1" {
  name     = "resource_user_1"
  enabled  = true
  keys     = "~key1*"
  channels = "&*"
  commands = "+@all"
}

resource "redisacl_user" "test2" {
  name     = "resource_user_2"
  enabled  = false
  keys     = "~key2*"
  channels = "&*"
  commands = "+@all"
}

data "redisacl_users" "test" {
  depends_on = [redisacl_user.test1, redisacl_user.test2]
}
`
}

func testAccACLUsersDataSourceConfig_empty() string {
	return `
provider "redisacl" {}

data "redisacl_users" "test" {}
`
}

func testAccACLUsersDataSourceConfig_userAttributes() string {
	return `
provider "redisacl" {}

data "redisacl_users" "test" {}
`
}