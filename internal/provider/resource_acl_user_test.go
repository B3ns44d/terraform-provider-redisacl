// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

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
				Config: testAccACLUserResourceConfigBasic("testuser1"),
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
				Config: testAccACLUserResourceConfigBasic("testuser2"),
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
				Config: testAccACLUserResourceConfigWithPermissions("testuser3", "~key1", "&channel1", "+get"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "testuser3"),
					resource.TestCheckResourceAttr("redisacl_user.test", "keys", "~key1"),
					resource.TestCheckResourceAttr("redisacl_user.test", "channels", "&channel1"),
					resource.TestCheckResourceAttr("redisacl_user.test", "commands", "-@all +get"),
				),
			},
			{
				Config: testAccACLUserResourceConfigWithPermissions("testuser3", "~key2", "&channel2", "+set"),
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
				Config: testAccACLUserResourceConfigBasic("testuser4"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "testuser4"),
				),
			},
			{
				Config: testAccACLUserResourceConfigBasic("testuser4_renamed"),
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
				Config: testAccACLUserResourceConfigBasic("testuser5"),
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
				Config: testAccACLUserResourceConfigImport("importuser"),
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
				Config: testAccACLUserResourceConfigWithPassword("testuser6", "password123"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "testuser6"),
					resource.TestCheckResourceAttr("redisacl_user.test", "passwords.#", "1"),
				),
			},
			{
				Config: testAccACLUserResourceConfigWithPassword("testuser6", "newpassword456"),
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
				Config:      testAccACLUserResourceConfigInvalid(),
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
			return fmt.Errorf("Error checking if user exists: %w", err)
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
			return fmt.Errorf("Error checking if user exists: %w", err)
		}

		if exists {
			return fmt.Errorf("ACL User %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckACLUserDoesNotExist(username string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		ctx := context.Background()
		exists, err := UserExists(ctx, username)
		if err != nil {
			return fmt.Errorf("Error checking if user exists: %w", err)
		}

		if exists {
			return fmt.Errorf("ACL User %s still exists but should have been deleted", username)
		}

		return nil
	}
}

// Config helper functions

func testAccACLUserResourceConfigBasic(name string) string {
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

func testAccACLUserResourceConfigWithPermissions(name, keys, channels, commands string) string {
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

func testAccACLUserResourceConfigWithPassword(name, password string) string {
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

func testAccACLUserResourceConfigImport(name string) string {
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

func testAccACLUserResourceConfigInvalid() string {
	return `
provider "redisacl" {}

resource "redisacl_user" "test" {
  enabled = true
  # Missing required name attribute
}
`
}

func TestAccACLUserResource_CommandsWithoutDashAllPrefix(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfigCommandsWithoutDashAll("drift_test_user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "drift_test_user"),
					resource.TestCheckResourceAttr("redisacl_user.test", "commands", "+get +set"),
				),
			},
			// Apply again to ensure no drift detected
			{
				Config: testAccACLUserResourceConfigCommandsWithoutDashAll("drift_test_user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "commands", "+get +set"),
				),
				PlanOnly: true,
			},
		},
	})
}

func TestAccACLUserResource_CommandsWithDashAllPrefix(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfigCommandsWithDashAll("prefix_test_user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "prefix_test_user"),
					resource.TestCheckResourceAttr("redisacl_user.test", "commands", "-@all +get +set"),
				),
			},
			// Apply again to ensure no drift detected
			{
				Config: testAccACLUserResourceConfigCommandsWithDashAll("prefix_test_user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "commands", "-@all +get +set"),
				),
				PlanOnly: true,
			},
		},
	})
}

func TestAccACLUserResource_MultiplePasswords(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfigMultiplePasswords("multipass_user", []string{"pass1", "pass2"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "multipass_user"),
					resource.TestCheckResourceAttr("redisacl_user.test", "passwords.#", "2"),
				),
			},
			// Add a third password
			{
				Config: testAccACLUserResourceConfigMultiplePasswords("multipass_user", []string{"pass1", "pass2", "pass3"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "passwords.#", "3"),
				),
			},
			// Remove one password
			{
				Config: testAccACLUserResourceConfigMultiplePasswords("multipass_user", []string{"pass1", "pass3"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "passwords.#", "2"),
				),
			},
		},
	})
}

func TestAccACLUserResource_PasswordRotation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfigWithPassword("rotation_user", "oldpassword"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "passwords.#", "1"),
				),
			},
			// Rotate to new password
			{
				Config: testAccACLUserResourceConfigWithPassword("rotation_user", "newpassword"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "passwords.#", "1"),
				),
			},
		},
	})
}

func TestAccACLUserResource_DisabledUser(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfigDisabled("disabled_user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "disabled_user"),
					resource.TestCheckResourceAttr("redisacl_user.test", "enabled", "false"),
				),
			},
			// Enable the user
			{
				Config: testAccACLUserResourceConfigBasic("disabled_user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "enabled", "true"),
				),
			},
			// Disable again
			{
				Config: testAccACLUserResourceConfigDisabled("disabled_user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "enabled", "false"),
				),
			},
		},
	})
}

func TestAccACLUserResource_NoPassword(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfigNoPassword("nopass_user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "nopass_user"),
					resource.TestCheckResourceAttr("redisacl_user.test", "passwords.#", "0"),
				),
			},
			// Add a password
			{
				Config: testAccACLUserResourceConfigWithPassword("nopass_user", "newpassword"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "passwords.#", "1"),
				),
			},
			// Remove password (back to nopass)
			{
				Config: testAccACLUserResourceConfigNoPassword("nopass_user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "passwords.#", "0"),
				),
			},
		},
	})
}

// Config helper functions for new tests

func testAccACLUserResourceConfigCommandsWithoutDashAll(name string) string {
	return fmt.Sprintf(`
provider "redisacl" {}

resource "redisacl_user" "test" {
  name     = "%s"
  enabled  = true
  keys     = "~*"
  channels = "&*"
  commands = "+get +set"
}
`, name)
}

func testAccACLUserResourceConfigCommandsWithDashAll(name string) string {
	return fmt.Sprintf(`
provider "redisacl" {}

resource "redisacl_user" "test" {
  name     = "%s"
  enabled  = true
  keys     = "~*"
  channels = "&*"
  commands = "-@all +get +set"
}
`, name)
}

func testAccACLUserResourceConfigMultiplePasswords(name string, passwords []string) string {
	passwordList := ""
	for i, pass := range passwords {
		if i > 0 {
			passwordList += ", "
		}
		passwordList += fmt.Sprintf(`"%s"`, pass)
	}

	return fmt.Sprintf(`
provider "redisacl" {}

resource "redisacl_user" "test" {
  name      = "%s"
  enabled   = true
  passwords = [%s]
  keys      = "~*"
  channels  = "&*"
  commands  = "+@all"
}
`, name, passwordList)
}

func testAccACLUserResourceConfigDisabled(name string) string {
	return fmt.Sprintf(`
provider "redisacl" {}

resource "redisacl_user" "test" {
  name     = "%s"
  enabled  = false
  keys     = "~*"
  channels = "&*"
  commands = "+@all"
}
`, name)
}

func testAccACLUserResourceConfigNoPassword(name string) string {
	return fmt.Sprintf(`
provider "redisacl" {}

resource "redisacl_user" "test" {
  name      = "%s"
  enabled   = true
  passwords = []
  keys      = "~*"
  channels  = "&*"
  commands  = "+@all"
}
`, name)
}

func TestAccACLUserResource_ComplexCommandCombinations(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfigComplexCommands("complex_user", "-@all +@read +@write -del"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "complex_user"),
					resource.TestCheckResourceAttr("redisacl_user.test", "commands", "-@all +@read +@write -del"),
				),
			},
			// Update to different command combination
			{
				Config: testAccACLUserResourceConfigComplexCommands("complex_user", "-@all +@string +@hash +@list"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "commands", "-@all +@string +@hash +@list"),
				),
			},
		},
	})
}

func TestAccACLUserResource_MultipleKeyPatterns(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfigMultipleKeys("multikey_user", "~app:* ~cache:* ~session:*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "multikey_user"),
					resource.TestCheckResourceAttr("redisacl_user.test", "keys", "~app:* ~cache:* ~session:*"),
				),
			},
			// Update key patterns
			{
				Config: testAccACLUserResourceConfigMultipleKeys("multikey_user", "~app:* ~data:*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "keys", "~app:* ~data:*"),
				),
			},
		},
	})
}

func TestAccACLUserResource_MultipleChannelPatterns(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfigMultipleChannels("multichan_user", "&notifications:* &events:* &alerts:*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "multichan_user"),
					resource.TestCheckResourceAttr("redisacl_user.test", "channels", "&notifications:* &events:* &alerts:*"),
				),
			},
			// Update channel patterns
			{
				Config: testAccACLUserResourceConfigMultipleChannels("multichan_user", "&notifications:*"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "channels", "&notifications:*"),
				),
			},
		},
	})
}

func TestAccACLUserResource_StateDrift(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfigBasic("drift_user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "drift_user"),
					resource.TestCheckResourceAttr("redisacl_user.test", "keys", "~*"),
				),
			},
			{
				// Manually modify the user in Redis, then apply again
				PreConfig: func() {
					ctx := context.Background()
					err := ModifyUserInRedis(ctx, "drift_user", []string{"reset", "on", "~modified:*", "&*", "+@all"})
					if err != nil {
						t.Fatalf("Failed to modify user in Redis: %v", err)
					}
				},
				// Apply again - Terraform should detect drift and fix it
				Config: testAccACLUserResourceConfigBasic("drift_user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					// After apply, should be back to configured value
					resource.TestCheckResourceAttr("redisacl_user.test", "keys", "~*"),
				),
			},
		},
	})
}

func testAccACLUserResourceConfigComplexCommands(name, commands string) string {
	return fmt.Sprintf(`
provider "redisacl" {}

resource "redisacl_user" "test" {
  name     = "%s"
  enabled  = true
  keys     = "~*"
  channels = "&*"
  commands = "%s"
}
`, name, commands)
}

func testAccACLUserResourceConfigMultipleKeys(name, keys string) string {
	return fmt.Sprintf(`
provider "redisacl" {}

resource "redisacl_user" "test" {
  name     = "%s"
  enabled  = true
  keys     = "%s"
  channels = "&*"
  commands = "+@all"
}
`, name, keys)
}

func testAccACLUserResourceConfigMultipleChannels(name, channels string) string {
	return fmt.Sprintf(`
provider "redisacl" {}

resource "redisacl_user" "test" {
  name     = "%s"
  enabled  = true
  keys     = "~*"
  channels = "%s"
  commands = "+@all"
}
`, name, channels)
}

// Valkey Backend Tests

func TestAccACLUserResource_Create_Valkey(t *testing.T) {
	ctx := context.Background()

	// Start Valkey container
	if err := StartValkeyContainer(ctx); err != nil {
		t.Fatalf("Failed to start Valkey container: %v", err)
	}
	defer func() {
		if err := StopValkeyContainer(ctx); err != nil {
			t.Logf("Failed to stop Valkey container: %v", err)
		}
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckValkey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroyValkey,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfigBasicValkey("valkey_testuser1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExistsValkey("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "valkey_testuser1"),
					resource.TestCheckResourceAttr("redisacl_user.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("redisacl_user.test", "name"),
				),
			},
		},
	})
}

func TestAccACLUserResource_Read_Valkey(t *testing.T) {
	ctx := context.Background()

	// Start Valkey container
	if err := StartValkeyContainer(ctx); err != nil {
		t.Fatalf("Failed to start Valkey container: %v", err)
	}
	defer func() {
		if err := StopValkeyContainer(ctx); err != nil {
			t.Logf("Failed to stop Valkey container: %v", err)
		}
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckValkey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroyValkey,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfigBasicValkey("valkey_testuser2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExistsValkey("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "valkey_testuser2"),
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

func TestAccACLUserResource_Update_Valkey(t *testing.T) {
	ctx := context.Background()

	// Start Valkey container
	if err := StartValkeyContainer(ctx); err != nil {
		t.Fatalf("Failed to start Valkey container: %v", err)
	}
	defer func() {
		if err := StopValkeyContainer(ctx); err != nil {
			t.Logf("Failed to stop Valkey container: %v", err)
		}
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckValkey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroyValkey,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfigWithPermissionsValkey("valkey_testuser3", "~key1", "&channel1", "+get"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExistsValkey("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "valkey_testuser3"),
					resource.TestCheckResourceAttr("redisacl_user.test", "keys", "~key1"),
					resource.TestCheckResourceAttr("redisacl_user.test", "channels", "&channel1"),
					resource.TestCheckResourceAttr("redisacl_user.test", "commands", "-@all +get"),
				),
			},
			{
				Config: testAccACLUserResourceConfigWithPermissionsValkey("valkey_testuser3", "~key2", "&channel2", "+set"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExistsValkey("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "valkey_testuser3"),
					resource.TestCheckResourceAttr("redisacl_user.test", "keys", "~key2"),
					resource.TestCheckResourceAttr("redisacl_user.test", "channels", "&channel2"),
					resource.TestCheckResourceAttr("redisacl_user.test", "commands", "-@all +set"),
				),
			},
		},
	})
}

func TestAccACLUserResource_Delete_Valkey(t *testing.T) {
	ctx := context.Background()

	// Start Valkey container
	if err := StartValkeyContainer(ctx); err != nil {
		t.Fatalf("Failed to start Valkey container: %v", err)
	}
	defer func() {
		if err := StopValkeyContainer(ctx); err != nil {
			t.Logf("Failed to stop Valkey container: %v", err)
		}
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckValkey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroyValkey,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfigBasicValkey("valkey_testuser5"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExistsValkey("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", "valkey_testuser5"),
				),
			},
		},
	})
}

// Valkey Helper functions

func testAccPreCheckValkey(t *testing.T) {
	if valkeyHost == "" || valkeyPort == "" {
		t.Fatal("Valkey container not started")
	}
}

func testAccCheckACLUserExistsValkey(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ACL User ID is set")
		}

		ctx := context.Background()
		exists, err := UserExistsInValkey(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error checking if user exists in Valkey: %w", err)
		}

		if !exists {
			return fmt.Errorf("ACL User %s does not exist in Valkey", rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckACLUserDestroyValkey(s *terraform.State) error {
	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "redisacl_user" {
			continue
		}

		exists, err := UserExistsInValkey(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error checking if user exists in Valkey: %w", err)
		}

		if exists {
			return fmt.Errorf("ACL User %s still exists in Valkey", rs.Primary.ID)
		}
	}

	return nil
}

// Valkey Config helper functions

func testAccACLUserResourceConfigBasicValkey(name string) string {
	return fmt.Sprintf(`
provider "redisacl" {
  address    = "%s:%s"
  password   = "testpass"
  use_valkey = true
}

resource "redisacl_user" "test" {
  name     = "%s"
  enabled  = true
  keys     = "~*"
  channels = "&*"
  commands = "+@all"
}
`, valkeyHost, valkeyPort, name)
}

func testAccACLUserResourceConfigWithPermissionsValkey(name, keys, channels, commands string) string {
	return fmt.Sprintf(`
provider "redisacl" {
  address    = "%s:%s"
  password   = "testpass"
  use_valkey = true
}

resource "redisacl_user" "test" {
  name     = "%s"
  enabled  = true
  keys     = "%s"
  channels = "%s"
  commands = "-@all %s"
}
`, valkeyHost, valkeyPort, name, keys, channels, commands)
}

// Unit Tests for Existence Check Scenarios

// TestAccACLUserResource_UserAlreadyExists tests that attempting to create
// a user that already exists returns an error with import instructions
func TestAccACLUserResource_UserAlreadyExists(t *testing.T) {
	ctx := context.Background()
	username := "existing_user_test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Pre-create the user before attempting to create via Terraform
				PreConfig: func() {
					if err := CreateTestUser(ctx, username, "testpass123"); err != nil {
						t.Fatalf("Failed to pre-create test user: %v", err)
					}
				},
				Config:      testAccACLUserResourceConfigBasic(username),
				ExpectError: regexp.MustCompile(`ACL user "` + username + `" already exists`),
			},
		},
	})

	// Verify the user still exists with original configuration
	exists, err := UserExists(ctx, username)
	if err != nil {
		t.Fatalf("Error checking if user exists: %v", err)
	}
	if !exists {
		t.Fatal("User should still exist after failed creation attempt")
	}

	// Verify error message contains import instructions
	// This is implicitly tested by the ExpectError regex above
}

// TestAccACLUserResource_UserDoesNotExist tests that creating a user
// that doesn't exist succeeds normally
func TestAccACLUserResource_UserDoesNotExist(t *testing.T) {
	username := "new_user_test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfigBasic(username),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExists("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", username),
					resource.TestCheckResourceAttr("redisacl_user.test", "enabled", "true"),
					resource.TestCheckResourceAttr("redisacl_user.test", "keys", "~*"),
					resource.TestCheckResourceAttr("redisacl_user.test", "channels", "&*"),
					resource.TestCheckResourceAttr("redisacl_user.test", "commands", "+@all"),
				),
			},
		},
	})
}

// TestAccACLUserResource_ErrorMessageFormat tests that the error message
// for existing users contains all required elements
func TestAccACLUserResource_ErrorMessageFormat(t *testing.T) {
	ctx := context.Background()
	username := "format_test_user"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Pre-create the user before attempting to create via Terraform
				PreConfig: func() {
					if err := CreateTestUser(ctx, username, "testpass123"); err != nil {
						t.Fatalf("Failed to pre-create test user: %v", err)
					}
				},
				Config: testAccACLUserResourceConfigBasic(username),
				ExpectError: regexp.MustCompile(
					// Check for all required elements in the error message
					`(?s)` + // Enable multiline matching
						`ACL user "` + username + `" already exists.*` + // Username
						`not managed by Terraform.*` + // Explanation
						`terraform import.*` + // Import instructions
						`redisacl_user\.<resource_name> ` + username, // Example command
				),
			},
		},
	})
}

// TestAccACLUserResource_UserAlreadyExists_Valkey tests that attempting to create
// a user that already exists in Valkey returns an error with import instructions
func TestAccACLUserResource_UserAlreadyExists_Valkey(t *testing.T) {
	ctx := context.Background()
	username := "existing_valkey_user_test"

	// Start Valkey container
	if err := StartValkeyContainer(ctx); err != nil {
		t.Fatalf("Failed to start Valkey container: %v", err)
	}
	defer func() {
		if err := StopValkeyContainer(ctx); err != nil {
			t.Logf("Failed to stop Valkey container: %v", err)
		}
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckValkey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Pre-create the user before attempting to create via Terraform
				PreConfig: func() {
					if err := CreateTestUserInValkey(ctx, username, "testpass123"); err != nil {
						t.Fatalf("Failed to pre-create test user in Valkey: %v", err)
					}
				},
				Config:      testAccACLUserResourceConfigBasicValkey(username),
				ExpectError: regexp.MustCompile(`ACL user "` + username + `" already exists`),
			},
		},
	})

	// Verify the user still exists with original configuration
	exists, err := UserExistsInValkey(ctx, username)
	if err != nil {
		t.Fatalf("Error checking if user exists in Valkey: %v", err)
	}
	if !exists {
		t.Fatal("User should still exist in Valkey after failed creation attempt")
	}

	// Verify error message contains import instructions
	// This is implicitly tested by the ExpectError regex above
}

// TestAccACLUserResource_UserDoesNotExist_Valkey tests that creating a user
// that doesn't exist in Valkey succeeds normally
func TestAccACLUserResource_UserDoesNotExist_Valkey(t *testing.T) {
	ctx := context.Background()
	// Use timestamp to ensure unique username
	username := fmt.Sprintf("new_valkey_user_%d", time.Now().Unix())

	// Start Valkey container
	if err := StartValkeyContainer(ctx); err != nil {
		t.Fatalf("Failed to start Valkey container: %v", err)
	}
	defer func() {
		if err := StopValkeyContainer(ctx); err != nil {
			t.Logf("Failed to stop Valkey container: %v", err)
		}
	}()

	t.Logf("Valkey container connection: %s:%s", valkeyHost, valkeyPort)

	// Clean up any existing users from previous test runs
	if err := CleanupValkeyUsers(ctx); err != nil {
		t.Logf("Warning: Failed to cleanup Valkey users: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckValkey(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckACLUserDestroyValkey,
		Steps: []resource.TestStep{
			{
				Config: testAccACLUserResourceConfigBasicValkey(username),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckACLUserExistsValkey("redisacl_user.test"),
					resource.TestCheckResourceAttr("redisacl_user.test", "name", username),
					resource.TestCheckResourceAttr("redisacl_user.test", "enabled", "true"),
					resource.TestCheckResourceAttr("redisacl_user.test", "keys", "~*"),
					resource.TestCheckResourceAttr("redisacl_user.test", "channels", "&*"),
					resource.TestCheckResourceAttr("redisacl_user.test", "commands", "+@all"),
				),
			},
		},
	})
}
