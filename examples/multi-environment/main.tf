# Multi-Environment Redis ACL Management
# This example shows how to manage Redis ACL users across different environments

terraform {
  required_providers {
    redisacl = {
      source  = "B3ns44d/redisacl"
      version = "~> 0.1.0"
    }
  }
}

# Variables for environment-specific configuration
variable "environments" {
  description = "Environment configurations"
  type = map(object({
    redis_address = string
    redis_password = string
    use_tls = bool
  }))
  default = {
    dev = {
      redis_address  = "localhost:6379"
      redis_password = "dev-password"
      use_tls        = false
    }
    staging = {
      redis_address  = "staging-redis.example.com:6379"
      redis_password = "staging-password"
      use_tls        = true
    }
    prod = {
      redis_address  = "prod-redis.example.com:6380"
      redis_password = "prod-password"
      use_tls        = true
    }
  }
}

variable "app_passwords" {
  description = "Application passwords for each environment"
  type = map(string)
  sensitive = true
  default = {
    dev     = "dev-app-password"
    staging = "staging-app-password"
    prod    = "prod-app-password"
  }
}

# Provider configurations for each environment
provider "redisacl" {
  alias    = "dev"
  address  = var.environments.dev.redis_address
  password = var.environments.dev.redis_password
  use_tls  = var.environments.dev.use_tls
}

provider "redisacl" {
  alias    = "staging"
  address  = var.environments.staging.redis_address
  password = var.environments.staging.redis_password
  use_tls  = var.environments.staging.use_tls
}

provider "redisacl" {
  alias    = "prod"
  address  = var.environments.prod.redis_address
  password = var.environments.prod.redis_password
  use_tls  = var.environments.prod.use_tls
}

# Local values for environment-specific configurations
locals {
  # Development: Permissive settings for easier debugging
  dev_permissions = {
    keys     = "~*"
    channels = "&*"
    commands = "+@all -@dangerous"
  }
  
  # Staging: More restrictive, similar to production
  staging_permissions = {
    keys     = "~app:* ~cache:* ~session:*"
    channels = "&app:* &notifications:*"
    commands = "+@read +@write +@stream -@dangerous -@admin"
  }
  
  # Production: Most restrictive settings
  prod_permissions = {
    keys     = "~app:* ~cache:*"
    channels = "&app:notifications:*"
    commands = "+@read +@write +@stream -@dangerous -@admin -@scripting"
  }
}

# Application users for each environment
resource "redisacl_user" "app_user_dev" {
  provider = redisacl.dev
  
  name      = "app-user"
  enabled   = true
  passwords = [var.app_passwords.dev]
  
  keys     = local.dev_permissions.keys
  channels = local.dev_permissions.channels
  commands = local.dev_permissions.commands
}

resource "redisacl_user" "app_user_staging" {
  provider = redisacl.staging
  
  name      = "app-user"
  enabled   = true
  passwords = [var.app_passwords.staging]
  
  keys     = local.staging_permissions.keys
  channels = local.staging_permissions.channels
  commands = local.staging_permissions.commands
}

resource "redisacl_user" "app_user_prod" {
  provider = redisacl.prod
  
  name      = "app-user"
  enabled   = true
  passwords = [var.app_passwords.prod]
  
  keys     = local.prod_permissions.keys
  channels = local.prod_permissions.channels
  commands = local.prod_permissions.commands
}

# Background job users (only in staging and production)
resource "redisacl_user" "job_user_staging" {
  provider = redisacl.staging
  
  name      = "job-processor"
  enabled   = true
  passwords = ["staging-job-password"]
  
  keys     = "~jobs:* ~temp:*"
  channels = "&job:*"
  commands = "+@read +@write +@list +@stream -@dangerous"
}

resource "redisacl_user" "job_user_prod" {
  provider = redisacl.prod
  
  name      = "job-processor"
  enabled   = true
  passwords = ["prod-job-password"]
  
  keys     = "~jobs:*"
  channels = "&job:notifications:*"
  commands = "+@read +@write +@list +@stream -@dangerous -@admin"
}

# Monitoring users for all environments
resource "redisacl_user" "monitoring_dev" {
  provider = redisacl.dev
  
  name      = "monitoring"
  enabled   = true
  passwords = ["dev-monitoring-password"]
  
  keys     = "~"  # No key access
  channels = "&"  # No channel access
  commands = "+ping +info +client +config|get +memory +latency +slowlog +dbsize"
}

resource "redisacl_user" "monitoring_staging" {
  provider = redisacl.staging
  
  name      = "monitoring"
  enabled   = true
  passwords = ["staging-monitoring-password"]
  
  keys     = "~"
  channels = "&"
  commands = "+ping +info +client +memory +latency +slowlog +dbsize"
}

resource "redisacl_user" "monitoring_prod" {
  provider = redisacl.prod
  
  name      = "monitoring"
  enabled   = true
  passwords = ["prod-monitoring-password"]
  
  keys     = "~"
  channels = "&"
  commands = "+ping +info +client +memory +latency +slowlog"
}

# Data sources to verify users in each environment
data "redisacl_users" "dev_users" {
  provider = redisacl.dev
}

data "redisacl_users" "staging_users" {
  provider = redisacl.staging
}

data "redisacl_users" "prod_users" {
  provider = redisacl.prod
}

# Outputs
output "environment_summary" {
  description = "Summary of users created in each environment"
  value = {
    dev = {
      total_users = length(data.redisacl_users.dev_users.users)
      user_names  = [for user in data.redisacl_users.dev_users.users : user.name]
    }
    staging = {
      total_users = length(data.redisacl_users.staging_users.users)
      user_names  = [for user in data.redisacl_users.staging_users.users : user.name]
    }
    prod = {
      total_users = length(data.redisacl_users.prod_users.users)
      user_names  = [for user in data.redisacl_users.prod_users.users : user.name]
    }
  }
}

output "app_user_permissions" {
  description = "Permissions for app users in each environment"
  value = {
    dev = {
      keys     = redisacl_user.app_user_dev.keys
      commands = redisacl_user.app_user_dev.commands
    }
    staging = {
      keys     = redisacl_user.app_user_staging.keys
      commands = redisacl_user.app_user_staging.commands
    }
    prod = {
      keys     = redisacl_user.app_user_prod.keys
      commands = redisacl_user.app_user_prod.commands
    }
  }
}