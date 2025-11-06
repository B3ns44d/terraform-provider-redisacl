# Basic Redis ACL User Management Example
# This example demonstrates basic user creation and management

terraform {
  required_providers {
    redisacl = {
      source  = "B3ns44d/redisacl"
      version = "1.0.1"
    }
  }
}

# Configure the provider for local Redis
provider "redisacl" {
  address  = "localhost:6379"
  username = "redis" # Change this to your Redis username
  password = "your-redis-password" # Change this to your Redis password
}

# Create a read-only user for applications
resource "redisacl_user" "readonly_app" {
  name      = "readonly-app"
  enabled   = true
  passwords = ["app-readonly-password"]

  # Allow access to application keys only
  keys = "~app:* ~cache:*"

  # Allow all pub/sub channels
  channels = "&*"

  # Only allow read operations
  commands = "+@read -@write -@dangerous"
}

# Create a write user for data ingestion
resource "redisacl_user" "write_app" {
  name      = "write-app"
  enabled   = true
  passwords = ["app-write-password"]

  # Allow access to specific key patterns
  keys = "~data:* ~temp:*"

  # Allow specific pub/sub channels
  channels = "&notifications:* &events:*"

  # Allow read and write, but not dangerous operations
  commands = "+@read +@write -@dangerous"
}

# Create an admin user with full access
resource "redisacl_user" "admin" {
  name      = "admin-user"
  enabled   = true
  passwords = ["secure-admin-password"]

  # Full access to all keys and channels
  keys     = "~*"
  channels = "&*"
  commands = "+@all"

  # Allow self-modification (needed for admin operations)
  allow_self_mutation = true
}

# Create a monitoring user with limited access
resource "redisacl_user" "monitoring" {
  name      = "monitoring-user"
  enabled   = true
  passwords = ["monitoring-password"]

  # No key access needed for monitoring
  keys = "~"

  # No pub/sub access needed
  channels = "&"

  # Only allow monitoring and info commands
  commands = "+ping +info +client +config|get +memory +latency +slowlog"
}

# Data source to read the default user
data "redisacl_user" "default" {
  name = "default"
}

# Data source to list all users
data "redisacl_users" "all" {}

# Outputs
output "default_user_info" {
  description = "Information about the default Redis user"
  value = {
    enabled  = data.redisacl_user.default.enabled
    keys     = data.redisacl_user.default.keys
    commands = data.redisacl_user.default.commands
  }
}

output "all_users" {
  description = "List of all Redis ACL users"
  value = [for user in data.redisacl_users.all.users : {
    name    = user.name
    enabled = user.enabled
  }]
}

output "created_users" {
  description = "Information about created users"
  value = {
    readonly_app = {
      name = redisacl_user.readonly_app.name
      keys = redisacl_user.readonly_app.keys
    }
    write_app = {
      name = redisacl_user.write_app.name
      keys = redisacl_user.write_app.keys
    }
    admin = {
      name = redisacl_user.admin.name
    }
    monitoring = {
      name = redisacl_user.monitoring.name
    }
  }
}