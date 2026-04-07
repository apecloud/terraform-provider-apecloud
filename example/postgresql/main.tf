terraform {
  required_providers {
    apecloud = {
      source = "registry.terraform.io/apecloud/apecloud"
    }
  }
}

provider "apecloud" {
  api_url = "https://api-dev.apecloud.cn"

  api_key    = "your_api_key"
  api_secret = "your_api_secret"

  admin_api_key    = "your_admin_api_key"
  admin_api_secret = "your_admin_api_secret"

  # if you need to skip verify, please set https_skip_verify = true
  # https_skip_verify = true
}

resource "apecloud_cluster" "my_pg" {
  name             = "my-pg"
  display_name     = "my-pg"
  org_name         = "my-org"
  environment_name = "prod"
  engine           = "postgresql"
  version          = "18.1.0"
  mode             = "replication" # or "standalone"
  cluster_type     = "Normal"
  project          = "kubeblocks-cloud-ns"

  single_zone        = true
  termination_policy = "Delete" # or "DoNotTerminate"
  maintaince_window = {
    start_hour = 18
    end_hour   = 22
    weekdays   = "1,2,3,4,5,6,7"
  }

  components = [
    {
      component = "postgresql"
      replicas  = 2
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
      class_code    = "postgresql.replication.postgresql.1c2g.general"
    }
  ]

  param_tpls = [
    {
      component           = "postgresql"
      param_tpl_name      = "postgresql-18.0-default-parameter-template"
      param_tpl_partition = "default"
    }
  ]

  backup = {
    auto_backup                 = true
    auto_backup_method          = "wal-g"
    backup_repo                 = "my-backuprepo"
    retention_period            = "7d"
    retention_policy            = "LastOne"
    cron_expression             = "0 18 * * *"
    snapshot_volumes            = false
    pitr_enabled                = true
    continuous_backup_method    = "wal-g-archive"
    incremental_backup_enabled  = false
    incremental_cron_expression = "0 18 * * *"
  }
}
