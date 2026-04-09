terraform {
  required_providers {
    kbcloud = {
      source = "registry.terraform.io/apecloud/kbcloud"
    }
  }
}

provider "kbcloud" {
  api_url        = "https://kb-cloud-apiserver-endpoint.com/api"

  api_key        = "your_api_key"
  api_secret     = "your_api_secret"

  admin_api_key  = "your_admin_api_key"
  admin_api_secret = "your_admin_api_secret"

  # if you need to skip verify, please set https_skip_verify = true
  # https_skip_verify = true
}


resource "kbcloud_cluster" "my_mongodb_sharding" {
  name             = "my-mongodb-sharding"
  display_name     = "my-mongodb-sharding"
  org_name         = "my-org"
  environment_name = "prod"
  engine           = "mongodb"
  version          = "8.0.17"
  mode             = "sharding"
  cluster_type     = "Normal"
  project          = "kubeblocks-cloud-ns"

  single_zone        = true
  termination_policy = "Delete"

  maintaince_window = {
    start_hour = 18
    end_hour   = 22
    weekdays   = "1,2,3,4,5,6,7"
  }

  components = [
    {
      component = "mongo-shard"
      comp_num  = 2
      replicas  = 3
      volumes = [
        {
          name    = "data"
          storage = 20 # GB
          io_limits = {
            read_iops  = 1000
            write_iops = 1000
          }
          io_reserves = {
            read_iops  = 1000
            write_iops = 1000
          }
        }
      ]
      storage_class = "apelocal-rawdisk-xfs"
      class_code    = "mongodb.sharding.mongo-shard.1c1g.general"
    },
    {
      component = "mongo-config-server"
      replicas  = 3
      volumes = [
        {
          name    = "data"
          storage = 20 # GB
          io_limits = {
            read_iops  = 1000
            write_iops = 1000
          }
          io_reserves = {
            read_iops  = 1000
            write_iops = 1000
          }
        }
      ]
      storage_class = "apelocal-rawdisk-xfs"
      class_code    = "mongodb.sharding.mongo-config-server.1c1g.general"
    },
    {
      component  = "mongo-mongos"
      replicas   = 2
      class_code = "mongodb.sharding.mongo-mongos.1c1g.general"
    }
  ]

  param_tpls = [
    {
      component           = "mongo-shard"
      param_tpl_name      = "mongo-shard-default-parameter-template"
      param_tpl_partition = "default"
    },
    {
      component           = "mongo-config-server"
      param_tpl_name      = "mongo-config-server-default-parameter-template"
      param_tpl_partition = "default"
    },
    {
      component           = "mongo-mongos"
      param_tpl_name      = "mongo-mongos-default-parameter-template"
      param_tpl_partition = "default"
    }
  ]

  backup = {
    auto_backup              = true
    auto_backup_method       = "pbm-physical"
    backup_repo              = "my-backuprepo"
    retention_period         = "7d"
    retention_policy         = "LastOne"
    cron_expression          = "0 18 * * *"
    snapshot_volumes         = false
    pitr_enabled             = true
    continuous_backup_method = "pbm-pitr"
  }
}

resource "kbcloud_cluster" "my_mongodb_replicaset" {
  name             = "my-mongodb-replicaset"
  display_name     = "my-mongodb-replicaset"
  org_name         = "my-org"
  environment_name = "prod"
  engine           = "mongodb"
  version          = "6.0.27"
  mode             = "replicaset"
  cluster_type     = "Normal"
  project          = "kubeblocks-cloud-ns"

  single_zone        = true
  termination_policy = "Delete"

  maintaince_window = {
    start_hour = 18
    end_hour   = 22
    weekdays   = "1,2,3,4,5,6,7"
  }

  components = [
    {
      component = "mongodb"
      replicas  = 3
      volumes = [
        {
          name    = "data"
          storage = 20 # GB
          io_limits = {
            read_iops  = 1000
            write_iops = 1000
          }
          io_reserves = {
            read_iops  = 1000
            write_iops = 1000
          }
        }
      ]
      storage_class = "apelocal-rawdisk-xfs"
      class_code    = "mongodb.replicaset.mongodb.1c1g.general"
    }
  ]

  param_tpls = [
    {
      component           = "mongodb"
      param_tpl_name      = "mongodb-default-parameter-template"
      param_tpl_partition = "default"
    }
  ]

  backup = {
    auto_backup              = true
    auto_backup_method       = "pbm-physical"
    backup_repo              = "my-backuprepo"
    retention_period         = "7d"
    retention_policy         = "LastOne"
    cron_expression          = "0 18 * * *"
    snapshot_volumes         = false
    pitr_enabled             = true
    continuous_backup_method = "pbm-pitr"
  }
}


resource "kbcloud_cluster" "my_mongodb_standalone" {
  name             = "my-mongodb-standalone"
  display_name     = "my-mongodb-standalone"
  org_name         = "my-org"
  environment_name = "prod"
  engine           = "mongodb"
  version          = "6.0.27"
  mode             = "standalone"
  cluster_type     = "Normal"
  project          = "kubeblocks-cloud-ns"

  single_zone        = true
  termination_policy = "Delete"

  maintaince_window = {
    start_hour = 18
    end_hour   = 22
    weekdays   = "1,2,3,4,5,6,7"
  }

  components = [
    {
      component = "mongodb"
      replicas  = 1
      volumes = [
        {
          name    = "data"
          storage = 20 # GB
          io_limits = {
            read_iops  = 1000
            write_iops = 1000
          }
          io_reserves = {
            read_iops  = 1000
            write_iops = 1000
          }
        }
      ]
      storage_class = "apelocal-rawdisk-xfs"
      class_code    = "mongodb.standalone.mongodb.1c1g.general"
    }
  ]

  param_tpls = [
    {
      component           = "mongodb"
      param_tpl_name      = "mongodb-default-parameter-template"
      param_tpl_partition = "default"
    }
  ]

  backup = {
    auto_backup              = true
    auto_backup_method       = "pbm-physical"
    backup_repo              = "my-backuprepo"
    retention_period         = "7d"
    retention_policy         = "LastOne"
    cron_expression          = "0 18 * * *"
    snapshot_volumes         = false
    pitr_enabled             = true
    continuous_backup_method = "pbm-pitr"
  }
}
