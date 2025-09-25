terraform {
  required_providers {
    redisacl = {
      source = "B3ns44d/redisacl"
    }
  }
}

provider "redisacl" {
  address = "localhost:6379"
}

resource "redisacl_user" "example" {
  name      = "example-user"
  enabled   = true
  passwords = ["password123"]
  keys      = "~*"
  channels  = "&*"
  commands  = "+@all -@dangerous"
}

data "redisacl_user" "example" {
  name = redisacl_user.example.name
}

data "redisacl_users" "all" {}

output "user_name" {
  value = data.redisacl_user.example.name
}

output "all_users" {
  value = data.redisacl_users.all.users
}