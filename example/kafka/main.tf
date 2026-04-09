terraform {
  required_providers {
    kbcloud = {
      source = "registry.terraform.io/apecloud/kbcloud"
    }
  }
}

provider "kbcloud" {
  api_url = "https://api-dev.apecloud.cn"

  api_key    = "your_api_key"
  api_secret = "your_api_secret"

  admin_api_key    = "your_admin_api_key"
  admin_api_secret = "your_admin_api_secret"

  # if you need to skip verify, please set https_skip_verify = true
  # https_skip_verify = true
}


resource "kbcloud_cluster" "my_combined_kafka" {
  name             = "my-combined-kafka"
  display_name     = "my-combined-kafka"
  org_name         = "my-org"
  environment_name = "prod"
  engine           = "kafka"
  version          = "3.9.0"
  mode             = "combined" # support "combined" "separated" "withZookeeper"
  cluster_type     = "Normal"
  project          = "kubeblocks-cloud-ns"

  single_zone        = true
  termination_policy = "Delete"
  network_mode       = "HeadlessService"

  extra = {
    sasl = {
      enable = true
    }
  }

  maintaince_window = {
    start_hour = 18
    end_hour   = 22
    weekdays   = "1,2,3,4,5,6,7"
  }

  components = [
    {
      component = "kafka-combine"
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
        },
        {
          name    = "metadata"
          storage = 5 # GB
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
      class_code    = "kafka.combined.kafka-combine.1c1g.general"
    }
  ]

  param_tpls = []

  backup = {
    auto_backup        = true
    auto_backup_method = "topics"
    backup_repo        = "my-backuprepo"
    retention_period   = "7d"
    retention_policy   = "LastOne"
    cron_expression    = "0 18 * * *"
    snapshot_volumes   = false
  }
}


resource "kbcloud_cluster" "my_separated_kafka" {
  name             = "my-separated-kafka"
  display_name     = "my-separated-kafka"
  org_name         = "my-org"
  environment_name = "prod"
  engine           = "kafka"
  version          = "3.9.0"
  mode             = "separated"
  cluster_type     = "Normal"
  project          = "kubeblocks-cloud-ns"

  single_zone        = true
  termination_policy = "Delete"
  network_mode       = "HeadlessService"

  extra = {
    sasl = {
      enable = true
    }
  }

  maintaince_window = {
    start_hour = 18
    end_hour   = 22
    weekdays   = "1,2,3,4,5,6,7"
  }

  components = [
    {
      component = "kafka-broker"
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
      class_code    = "kafka.separated.kafka-broker.1c1g.general"
    },
    {
      component = "kafka-controller"
      replicas  = 3
      volumes = [
        {
          name    = "metadata"
          storage = 5 # GB
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
      class_code    = "kafka.separated.kafka-controller.1c1g.general"
    }
  ]

  backup = {
    auto_backup        = true
    auto_backup_method = "topics"
    backup_repo        = "my-backuprepo"
    retention_period   = "7d"
    retention_policy   = "LastOne"
    cron_expression    = "0 18 * * *"
    snapshot_volumes   = false
  }
}


resource "kbcloud_cluster" "my_zookeeper_kafka" {
  name             = "my-zookeeper-kafka"
  display_name     = "my-zookeeper-kafka"
  org_name         = "my-org"
  environment_name = "prod"
  engine           = "kafka"
  version          = "2.8.2"
  mode             = "withZookeeper-10"
  cluster_type     = "Normal"
  project          = "kubeblocks-cloud-ns"

  single_zone        = true
  termination_policy = "Delete"
  network_mode       = "HeadlessService"

  extra = {
    sasl = {
      enable = true
    }
  }

  service_refs = [
    {
      name    = "zookeeper"
      cluster = "oak24"
    }
  ]

  maintaince_window = {
    start_hour = 18
    end_hour   = 22
    weekdays   = "1,2,3,4,5,6,7"
  }

  components = [
    {
      component = "kafka-broker"
      replicas  = 5
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
        },
        {
          name    = "metadata"
          storage = 5 # GB
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
      class_code    = "kafka.withZookeeper-10.kafka-broker.1c1g.general"
    }
  ]

  backup = {
    auto_backup        = true
    auto_backup_method = "topics"
    backup_repo        = "my-backuprepo"
    retention_period   = "7d"
    retention_policy   = "LastOne"
    cron_expression    = "0 18 * * *"
    snapshot_volumes   = false
  }
}
