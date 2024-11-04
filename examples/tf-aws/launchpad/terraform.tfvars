// used to name infrastructure (CHANGE THIS)
name = "mcc-smoke-test"
aws = {
  region = "us-east-1"
}

launchpad = {
  drain = false

  mcr_version = "23.0.15"
  mke_version = "3.7.15"
  msr_version = ""

  mke_connect = {
    username = "admin"
    password = "m!rantis2024"
    insecure = false
  }
}

// configure the network stack
network = {
  cidr = "172.31.0.0/16"
}
subnets = {
  "Main" = {
    cidr       = "172.31.0.0/17"
    nodegroups = ["ACon", "AWrk_Ubu22", "AWrk_Roc9", "AWrk_Win2022"]
    private    = false
  }
}


// machine node groups by role & platform
nodegroups = {
  "ACon" = { // managers for A group
    role     = "manager"
    platform = "ubuntu_22.04"
    count    = 1
    type     = "m6a.2xlarge"
  },
  "AWrk_Ubu22" = { // workers for A group
    role        = "worker"
    platform    = "ubuntu_22.04"
    count       = 1
    type        = "c6a.xlarge"
    volume_size = 100
  },
  "AWrk_Roc9" = { // workers for A group
    role        = "worker"
    platform    = "rocky_9"
    count       = 1
    type        = "c6a.xlarge"
    volume_size = 100
  },
  //  "AWrk_Win2022" = {
  //    role        = "worker"
  //    platform   = "windows_core_2022"
  //    count       = 1
  //    type        = "c6a.xlarge"
  //  },
}

// set a windows password, if you have windows nodes
# windows_password = "testp@ss!"
