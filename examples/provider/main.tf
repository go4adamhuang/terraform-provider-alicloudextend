terraform {
  required_providers {
    alicloud = {
      source = "aliyun/alicloud"
    }
    alicloudextend = {
      source  = "go4adamhuang/alicloudextend"
      version = "~> 1.0"
    }
  }
}

provider "alicloudextend" {
  profile = "default"
}
