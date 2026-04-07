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

resource "apecloud_cluster" "my_replication_redis" {
  name             = "my-redis"
  display_name     = "my-redis"
  org_name         = "my-org"
  environment_name = "prod"
  engine           = "redis"
  version          = "7.2.12"
  mode             = "replication" # or "standalone" "cluster" "sentinel"
  cluster_type     = "Normal"
  project          = "kubeblocks-cloud-ns"

  single_zone        = true
  termination_policy = "Delete" # or ""
  maintaince_window = {
    start_hour = 18
    end_hour   = 22
    weekdays   = "1,2,3,4,5,6,7"
  }

  components = [
    {
      component = "redis"
      replicas  = 2
      volumes = [
        {
          name    = "data"
          storage = 30 # GB
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
      class_code    = "redis.replication.redis.2c2g.general" # check your class code
    }
  ]

  param_tpls = [
    {
      component           = "redis"
      param_tpl_name      = "redis-default-parameter-template"
      param_tpl_partition = "default"
      # if you want tu use param_tpl_id, please query by api first
      # param_tpl_id        = "485515016036942434" 
    }
  ]

  backup = {
    auto_backup              = true
    auto_backup_method       = "datafile"
    backup_repo              = "my-backuprepo"
    retention_period         = "7d"
    retention_policy         = "LastOne"
    cron_expression          = "0 18 * * *"
    snapshot_volumes         = false
    pitr_enabled             = true
    continuous_backup_method = "aof"
  }
}


resource "apecloud_cluster" "my_cluster_redis" {
  name             = "my-cluster-redis"
  display_name     = "my-cluster-redis"
  org_name         = "my-org"
  environment_name = "prod"
  engine           = "redis"
  version          = "7.2.12"
  mode             = "cluster" # or "standalone" "replication" "sentinel"
  cluster_type     = "Normal"
  project          = "kubeblocks-cloud-ns"

  single_zone        = true
  termination_policy = "DoNotTerminate" # or "Delete"
  network_mode       = "HeadlessService"
  maintaince_window = {
    start_hour = 18
    end_hour   = 22
    weekdays   = "1,2,3,4,5,6,7"
  }

  components = [
    {
      component = "redis-cluster"
      comp_num  = 3 # shard num
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
      class_code    = "redis.cluster.redis-cluster.2c1g.general" # check your class code
    }
  ]

  param_tpls = [
    {
      component           = "redis-cluster"
      param_tpl_name      = "redis-cluster-default-parameter-template"
      param_tpl_partition = "default"
    }
  ]

  backup = {
    auto_backup              = true
    auto_backup_method       = "datafile"
    backup_repo              = "my-backuprepo"
    retention_period         = "7d"
    retention_policy         = "LastOne"
    cron_expression          = "0 18 * * *"
    snapshot_volumes         = false
    continuous_backup_method = "aof"
  }
}

resource "apecloud_cluster" "my_sentinel_redis" {
  name             = "my-sentinel-redis"
  display_name     = "my-sentinel-redis"
  org_name         = "my-org"
  environment_name = "prod"
  engine           = "redis"
  version          = "7.2.12"
  mode             = "sentinel"
  cluster_type     = "Normal"
  project          = "kubeblocks-cloud-ns"

  single_zone        = true
  termination_policy = "Delete" # or "DoNotTerminate"
  network_mode       = "HostNetwork"

  extra = {
    sentinel = "{}"
  }

  maintaince_window = {
    start_hour = 18
    end_hour   = 22
    weekdays   = "1,2,3,4,5,6,7"
  }

  components = [
    {
      component = "redis"
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
      class_code    = "redis.sentinel.redis.2c1g.general"
    },
    {
      component = "redis-sentinel"
      replicas  = 3
      volumes = [
        {
          name    = "data"
          storage = 20 # GB
          io_limits = {
            read_iops  = 500
            write_iops = 500
          }
          io_reserves = {
            read_iops  = 500
            write_iops = 500
          }
        }
      ]
      storage_class = "apelocal-rawdisk-xfs"
      class_code    = "redis.sentinel.redis-sentinel.0.5c0.5g.general"
    }
  ]

  param_tpls = [
    {
      component           = "redis"
      param_tpl_name      = "redis-default-parameter-template"
      param_tpl_partition = "default"
    }
  ]

  backup = {
    auto_backup              = true
    auto_backup_method       = "datafile"
    backup_repo              = "my-backuprepo"
    retention_period         = "7d"
    retention_policy         = "LastOne"
    cron_expression          = "0 18 * * *"
    snapshot_volumes         = false
    pitr_enabled             = false
    continuous_backup_method = "aof"
  }
}
