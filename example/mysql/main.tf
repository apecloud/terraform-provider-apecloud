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

resource "apecloud_cluster" "my_mysql" {
  name               = "my-mysql"
  org_name           = "my-org"
  environment_name   = "prod"
  engine             = "mysql"
  version            = "8.0.44"
  mode               = "replication"
  cluster_type       = "Normal"
  project            = "kubeblocks-cloud-ns"
  single_zone        = true
  termination_policy = "Delete"
  maintaince_window = {
    start_hour = 22
    end_hour   = 18
    weekdays   = "1,2,3,4,5,6,7"
  }
  param_tpls = [
    {
      component           = "mysql"
      param_tpl_name      = "mysql-8.0-default-parameter-template"
      param_tpl_partition = "default"
    }
  ]

  init_options = [
    {
      component = "mysql"
      init_params = {
        "default-time-zone"      = "+08:00"
        "lower_case_table_names" = "1"
      }
      spec_name = "mysql-replication-config"
    }
  ]

  components = [
    {
      component = "mysql"
      replicas  = 2
      volumes = [
        {
          name    = "data"
          storage = 20 # GB
        }
      ]
      storage_class = "my-storage-class"
      class_code    = "mysql.replication.mysql.1c2g.general"
    }
  ]

  backup = {
    backup_repo      = "my-backuprepo"
    retention_policy = "LastOne"
    snapshot_volumes = false
  }
}
