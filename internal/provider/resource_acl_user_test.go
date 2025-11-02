// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start Redis container
	if err := StartRedisContainer(ctx); err != nil {
		fmt.Printf("Failed to start Redis container: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	if err := StopRedisContainer(ctx); err != nil {
		fmt.Printf("Failed to stop Redis container: %v\n", err)
	}

	os.Exit(code)
}

func TestAccACLUserResource_Create(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfig_basic("testuser1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "testuser1"),
					resource.TestCheckResourceAttr("redisacl_user.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("redisacl_user.test", "name"),
				),
			},
		},
	})
}

func TestAccACLUserResource_Read(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfig_basic("testuser2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "testuser2"),
					resource.TestCheckResourceAttr("redisacl_user.test", "enabled", "true"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "redisacl_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"passwords", "allow_self_mutation"},
			},
		},
	})
}

func TestAccACLUserResource_Update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfig_withPermissions("testuser3", "~key1", "&channel1", "+get"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "testuser3"),
					resource.TestCheckResourceAttr("redisacl_user.test", "keys", "~key1"),
					resource.TestCheckResourceAttr("redisacl_user.test", "channels", "&channel1"),
					resource.TestCheckResourceAttr("redisacl_user.test", "commands", "-@all +get"),
				),
			},
			{
				Config: testAccACLUserResourceConfig_withPermissions("testuser3", "~key2", "&channel2", "+set"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "testuser3"),
					resource.TestCheckResourceAttr("redisacl_user.test", "keys", "~key2"),
					resource.TestCheckResourceAttr("redisacl_user.test", "channels", "&channel2"),
					resource.TestCheckResourceAttr("redisacl_user.test", "commands", "-@all +set"),
				),
			},
		},
	})
}

func TestAccACLUserResource_ForceReplaceOnNameChange(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfig_basic("testuser4"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "testuser4"),
				),
			},
			{
				Config: testAccACLUserResourceConfig_basic("testuser4_renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "testuser4_renamed"),
					// Verify old user no longer exists
					testAccCheckACLUserDoesNotExist("testuser4"),
				),
			},
		},
	})
}

func TestAccACLUserResource_Delete(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfig_basic("testuser5"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "testuser5"),
				),
			},
		},
	})
}

func TestAccACLUserResource_ImportState(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			// First create the resource normally
			{
				Config: testAccACLUserResourceConfig_import("importuser"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.import_test"),
					resource.TestCheckResourceAttr("redisacl_user.import_test", "name", "importuser"),
				),
			},
			// Then test import
			{
				ResourceName:            "redisacl_user.import_test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"passwords", "allow_self_mutation"},
			},
		},
	})
}

func TestAccACLUserResource_WithPassword(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfig_withPassword("testuser6", "password123"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "testuser6"),
					resource.TestCheckResourceAttr("redisacl_user.test", "passwords.#", "1"),
				),
			},
			{
				Config: testAccACLUserResourceConfig_withPassword("testuser6", "newpassword456"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "testuser6"),
					resource.TestCheckResourceAttr("redisacl_user.test", "passwords.#", "1"),
				),
			},
		},
	})
}

func TestAccACLUserResource_InvalidConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccACLUserResourceConfig_invalid(),
				ExpectError: regexp.MustCompile("Missing required argument"),
			},
		},
	})
}

// Helper functions

func testAccCheckACLUserExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ACL User ID is set")
		}

		ctx := context.Background()
		exists, err := UserExists(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error checking if user exists: %v", err)
		}

		if !exists {
			return fmt.Errorf("ACL User %s does not exist", rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckACLUserDestroy(s *terraform.State) error {
	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "redisacl_user" {
			continue
		}

		exists, err := UserExists(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error checking if user exists: %v", err)
		}

		if exists {
			return fmt.Errorf("ACL User %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckACLUserDoesNotExist(username string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ctx := context.Background()
		exists, err := UserExists(ctx, username)
		if err != nil {
			return fmt.Errorf("Error checking if user exists: %v", err)
		}

		if exists {
			return fmt.Errorf("ACL User %s still exists but should have been deleted", username)
		}

		return nil
	}
}

// Config helper functions

func testAccACLUserResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
provider "redisacl" {}

resource "redisacl_user" "test" {
  name     = "%s"
  enabled  = true
  keys     = "~*"
  channels = "&*"
  commands = "+@all"
}
`, name)
}

func testAccACLUserResourceConfig_withPermissions(name, keys, channels, commands string) string {
	return fmt.Sprintf(`
provider "redisacl" {}

resource "redisacl_user" "test" {
  name     = "%s"
  enabled  = true
  keys     = "%s"
  channels = "%s"
  commands = "-@all %s"
}
`, name, keys, channels, commands)
}

func testAccACLUserResourceConfig_withPassword(name, password string) string {
	return fmt.Sprintf(`
provider "redisacl" {}

resource "redisacl_user" "test" {
  name      = "%s"
  enabled   = true
  passwords = ["%s"]
  keys      = "~*"
  channels  = "&*"
  commands  = "+@all"
}
`, name, password)
}

func testAccACLUserResourceConfig_import(name string) string {
	return fmt.Sprintf(`
provider "redisacl" {}

resource "redisacl_user" "import_test" {
  name     = "%s"
  enabled  = true
  keys     = "~*"
  channels = "&*"
  commands = "+@all"
}
`, name)
}

func testAccACLUserResourceConfig_invalid() string {
	return `
provider "redisacl" {}

resource "redisacl_user" "test" {
  enabled = true
  # Missing required name attribute
}
`
}
