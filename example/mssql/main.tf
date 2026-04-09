terraform {
  required_providers {
    kbcloud = {
      source = "registry.terraform.io/apecloud/kbcloud"
    }
  }
}

provider "kbcloud" {
  # api_url        = "https://api-dev.apecloud.cn"
  api_url = "https://api-dev.apecloud.cn"
  #   api_url        = "http://127.0.0.1:8080"

  # api_key        = "your_api_key"
  # api_secret     = "your_api_secret"

  # admin_api_key  = "your_admin_api_key"
  # admin_api_secret = "your_admin_api_secret"
  api_key    = "67a58b4e-c734-44f8-aa14-b8a6e4fe233d"
  api_secret = "696f77746b716d6a6475626a63766e6d646c7a65796d776a72746a6967747271"

  # admin_api_key  = "e06e6996-1afa-4ab4-8ed7-82a85747a1e3"
  # admin_api_secret = "6e70686872737663676470727773727463747164656e6e6b626f7a786c726b7a"
  # if you need to skip verify, please set https_skip_verify = true
  # https_skip_verify = true
}

resource "kbcloud_cluster" "my_mssql" {
  name             = "my-mssql-cluster"
  display_name     = "my-mssql-cluster"
  org_name         = "my-org"
  environment_name = "prod"
  engine           = "mssql"
  version          = "2022.19.0"
  mode             = "cluster"
  cluster_type     = "Normal"
  project          = "kubeblocks-cloud-ns"

  single_zone        = true
  termination_policy = "Delete"

  extra = {
    certificate    = {
      custom = false
    }
    collation      = "Chinese_PRC_CI_AS"
    defaultDBName  = "db1"
    productEdition = "Enterprise"
  }

  maintaince_window = {
    start_hour = 18
    end_hour   = 22
    weekdays   = "1,2,3,4,5,6,7"
  }

  components = [
    {
      component = "mssql"
      replicas  = 3
      volumes = [
        {
          name    = "data"
          storage = 20 # GB
          io_limits = {
            read_iops  = 2000
            write_iops = 2000
          }
          io_reserves = {
            read_iops  = 2000
            write_iops = 2000
          }
        }
      ]
      storage_class = "apelocal-rawdisk-xfs"
      class_code    = "mssql.cluster.mssql.2c4g.general"
    }
  ]

  backup = {
    auto_backup                 = true
    auto_backup_method          = "full"
    backup_repo                 = "my-backuprepo"
    retention_period            = "7d"
    retention_policy            = "LastOne"
    cron_expression             = "0 18 * * *"
    snapshot_volumes            = false
    pitr_enabled                = true
    continuous_backup_method    = "transaction-log"
    incremental_backup_enabled  = false
    incremental_cron_expression = "0 18 * * *"
  }
}
