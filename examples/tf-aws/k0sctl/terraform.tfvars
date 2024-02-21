// used to name infrastructure (CHANGE THIS)
name = "k0s-test"

aws = {
  region = "eu-central-1"
}

k0sctl = {
  version = "v1.28.4+k0s.0"

  no_wait  = false
  no_drain = false

  force = false

  disable_downgrade_check = false

  restore_from = ""
}

// configure the network stack
network = {
  cidr                 = "172.31.0.0/16"
  public_subnet_count  = 3
  private_subnet_count = 0 // if 0 then no private nodegroups allowed
}

// one definition for each group of machines to include in the stack
nodegroups = {
  "ACon" = { // controllers for A
    platform    = "ubuntu_22.04"
    count       = 1
    type        = "m6a.2xlarge"
    volume_size = 100
    role        = "controller"
    public      = true
    user_data   = ""
  },
  "AWrk_Ubu22" = {
    platform    = "ubuntu_22.04"
    count       = 1
    type        = "c6a.xlarge"
    volume_size = 100
    public      = true
    role        = "worker"
    user_data   = ""
  }
}
