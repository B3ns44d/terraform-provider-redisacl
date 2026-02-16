
terraform {
  required_providers {
    redisacl = {
      source  = "B3ns44d/redisacl"
      version = "1.0.1"
    }
  }
}

provider "redisacl" {
  username = "redis"
  password = ""
  use_tls  = false
  sentinel = {
    master_name = "mymaster"
    addresses = ["localhost:26111", "localhost:26112", "localhost:26113"]
    sync_acls_to_replicas = true  # New option
    username = "redis"
    password = ""
  }
}

resource "redisacl_user" "app" {
  name     = "app-user"
  enabled  = true
  passwords = ["secure-pass"]
  # ACL will be synced to all replicas automatically
}