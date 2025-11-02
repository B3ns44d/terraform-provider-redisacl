# TLS and mTLS Redis ACL Example
# This example demonstrates secure Redis connections with TLS and mutual TLS

terraform {
  required_providers {
    redisacl = {
      source  = "B3ns44d/redisacl"
      version = "~> 0.1.0"
    }
  }
}

# Variables for TLS configuration
variable "redis_tls_config" {
  description = "TLS configuration for Redis connections"
  type = object({
    address              = string
    password             = string
    ca_cert_path         = string
    client_cert_path     = string
    client_key_path      = string
    insecure_skip_verify = bool
  })
  default = {
    address              = "secure-redis.example.com:6380"
    password             = "secure-redis-password"
    ca_cert_path         = "certs/ca.pem"
    client_cert_path     = "certs/client.crt"
    client_key_path      = "certs/client.key"
    insecure_skip_verify = false
  }
}

# Provider with basic TLS (server authentication only)
provider "redisacl" {
  alias    = "tls_basic"
  address  = var.redis_tls_config.address
  password = var.redis_tls_config.password
  use_tls  = true

  # For testing with self-signed certificates (NOT for production)
  tls_insecure_skip_verify = var.redis_tls_config.insecure_skip_verify
}

# Provider with mutual TLS (client and server authentication)
provider "redisacl" {
  alias    = "mtls"
  address  = var.redis_tls_config.address
  password = var.redis_tls_config.password
  use_tls  = true

  # CA certificate to verify server
  tls_ca_cert = file(var.redis_tls_config.ca_cert_path)

  # Client certificate and key for mutual authentication
  tls_cert = file(var.redis_tls_config.client_cert_path)
  tls_key  = file(var.redis_tls_config.client_key_path)
}

# Provider for Redis Sentinel with TLS
provider "redisacl" {
  alias    = "sentinel_tls"
  username = "default"
  password = "master-password"
  use_tls  = true

  sentinel {
    master_name = "mymaster"
    addresses = [
      "sentinel1.example.com:26380",
      "sentinel2.example.com:26380",
      "sentinel3.example.com:26380"
    ]
    username = "sentinel-user"
    password = "sentinel-password"
  }

  tls_ca_cert = file(var.redis_tls_config.ca_cert_path)
}

# Provider for Redis Cluster with TLS
provider "redisacl" {
  alias    = "cluster_tls"
  username = "cluster-user"
  password = "cluster-password"
  use_tls  = true

  cluster {
    addresses = [
      "cluster-node1.example.com:6380",
      "cluster-node2.example.com:6380",
      "cluster-node3.example.com:6380"
    ]
  }

  tls_ca_cert = file(var.redis_tls_config.ca_cert_path)
  tls_cert    = file(var.redis_tls_config.client_cert_path)
  tls_key     = file(var.redis_tls_config.client_key_path)
}

# Application user with basic TLS
resource "redisacl_user" "app_user_tls" {
  provider = redisacl.tls_basic

  name      = "app-tls-user"
  enabled   = true
  passwords = ["secure-app-password"]

  keys     = "~app:* ~cache:*"
  channels = "&app:*"
  commands = "+@read +@write -@dangerous"
}

# High-security user with mutual TLS
resource "redisacl_user" "secure_user_mtls" {
  provider = redisacl.mtls

  name      = "secure-mtls-user"
  enabled   = true
  passwords = ["very-secure-password"]

  keys     = "~secure:* ~encrypted:*"
  channels = "&secure:*"
  commands = "+@read +@write +@stream -@dangerous -@admin"
}

# Admin user for Sentinel setup
resource "redisacl_user" "sentinel_admin" {
  provider = redisacl.sentinel_tls

  name      = "sentinel-admin"
  enabled   = true
  passwords = ["sentinel-admin-password"]

  keys     = "~*"
  channels = "&*"
  commands = "+@all"

  allow_self_mutation = true
}

# Cluster application user
resource "redisacl_user" "cluster_app_user" {
  provider = redisacl.cluster_tls

  name      = "cluster-app-user"
  enabled   = true
  passwords = ["cluster-app-password"]

  keys     = "~{app}:* ~{cache}:*" # Hash tags for cluster
  channels = "&app:*"
  commands = "+@read +@write +@stream -@dangerous"
}

# Monitoring user with TLS (minimal permissions)
resource "redisacl_user" "monitoring_tls" {
  provider = redisacl.tls_basic

  name      = "monitoring-tls"
  enabled   = true
  passwords = ["monitoring-password"]

  keys     = "~"
  channels = "&"
  commands = "+ping +info +client +memory +latency"
}

# Data sources to verify TLS connections work
data "redisacl_users" "tls_users" {
  provider = redisacl.tls_basic
}

data "redisacl_users" "mtls_users" {
  provider = redisacl.mtls
}

# Outputs
output "tls_connection_summary" {
  description = "Summary of TLS connections and users"
  value = {
    basic_tls = {
      total_users = length(data.redisacl_users.tls_users.users)
      user_names  = [for user in data.redisacl_users.tls_users.users : user.name]
    }
    mutual_tls = {
      total_users = length(data.redisacl_users.mtls_users.users)
      user_names  = [for user in data.redisacl_users.mtls_users.users : user.name]
    }
  }
}

output "security_configurations" {
  description = "Security configurations for each user type"
  value = {
    app_user_tls = {
      name     = redisacl_user.app_user_tls.name
      keys     = redisacl_user.app_user_tls.keys
      commands = redisacl_user.app_user_tls.commands
      security = "TLS encrypted connection"
    }
    secure_user_mtls = {
      name     = redisacl_user.secure_user_mtls.name
      keys     = redisacl_user.secure_user_mtls.keys
      commands = redisacl_user.secure_user_mtls.commands
      security = "Mutual TLS with client certificates"
    }
    cluster_app_user = {
      name     = redisacl_user.cluster_app_user.name
      keys     = redisacl_user.cluster_app_user.keys
      security = "TLS cluster with hash tags"
    }
  }
}

# Certificate validation output (for debugging)
output "certificate_info" {
  description = "Information about certificate files (for debugging)"
  value = {
    ca_cert_exists     = fileexists(var.redis_tls_config.ca_cert_path)
    client_cert_exists = fileexists(var.redis_tls_config.client_cert_path)
    client_key_exists  = fileexists(var.redis_tls_config.client_key_path)
  }
}