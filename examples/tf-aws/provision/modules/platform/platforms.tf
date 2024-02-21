
locals {
  // this is the idea of @jcarrol who put this into a lib map. Here we steal his idea
  lib_platform_definitions = {
    "centos_7" : {
      "ami_name" : "CentOS Linux 7 x86_64*",
      "owner" : "125523088429",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "centos",
      "ssh_port" : 22
    },
    "centos_7.9" : {
      "ami_name" : "CentOS Linux 7 x86_64*",
      "owner" : "125523088429",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "centos",
      "ssh_port" : 22
    },
    "oracle_7" : {
      "ami_name" : "OL7.?-x86_64-HVM-*",
      "owner" : "131827586825",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "oracle_7.6" : {
      "ami_name" : "OL7.6-x86_64-HVM-*",
      "owner" : "131827586825",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "oracle_7.8" : {
      "ami_name" : "OL7.8-x86_64-HVM-*",
      "owner" : "131827586825",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "oracle_7.9" : {
      "ami_name" : "OL7.9-x86_64-HVM-*",
      "owner" : "131827586825",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "oracle_8.2" : {
      "ami_name" : "OL8.2-x86_64-HVM-2020-05-22",
      "owner" : "131827586825",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "oracle_8.3" : {
      "ami_name" : "OL8.3-x86_64-HVM-*",
      "owner" : "131827586825",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "oracle_8.6" : {
      "ami_name" : "OL8.6-x86_64-HVM-*",
      "owner" : "131827586825",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "oracle_9.0" : {
      "ami_name" : "OL9.0-x86_64-HVM-*",
      "owner" : "131827586825",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "oracle_9.1" : {
      "ami_name" : "OL9.1-x86_64-HVM-*",
      "owner" : "131827586825",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "oracle_9.2" : {
      "ami_name" : "OL9.2-x86_64-HVM-*",
      "owner" : "131827586825",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "oracle_9" : {
      "ami_name" : "OL9.2-x86_64-HVM-*",
      "owner" : "131827586825",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_7" : {
      "ami_name" : "RHEL-7.?_HVM*x86_64*Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_7.5" : {
      "ami_name" : "RHEL-7.5_HVM*x86_64*Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_7.6" : {
      "ami_name" : "RHEL-7.6_HVM*x86_64*Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_7.7" : {
      "ami_name" : "RHEL-7.7_HVM*x86_64*Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_7.8" : {
      "ami_name" : "RHEL-7.8_HVM*x86_64*Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_7.9" : {
      "ami_name" : "RHEL-7.9_HVM*x86_64*Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_8.0" : {
      "ami_name" : "RHEL-8.0_HVM-201?????-x86_64-*-Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_8.1" : {
      "ami_name" : "RHEL-8.1.0_HVM-201?????-x86_64-*-Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_8.2" : {
      "ami_name" : "RHEL-8.2_HVM-202?????-x86_64-*-Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_8.3" : {
      "ami_name" : "RHEL-8.3*HVM-202?????-x86_64-*-Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_8.4" : {
      "ami_name" : "RHEL-8.4*HVM-202?????-x86_64-*-Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_8.5" : {
      "ami_name" : "RHEL-8.5*HVM-202?????-x86_64-*-Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_8.6" : {
      "ami_name" : "RHEL-8.6*HVM-202?????-x86_64-*-Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_8.7" : {
      "ami_name" : "RHEL-8.7*HVM-202?????-x86_64-*-Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_8.8" : {
      "ami_name" : "RHEL-8.8*HVM-202?????-x86_64-*-Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_8.9" : {
      "ami_name" : "RHEL-8.9.?_HVM-202?????-x86_64-*-Hourly2-GP?",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_8" : {
      "ami_name" : "RHEL-8.8*HVM-202?????-x86_64-*-Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_9.0" : {
      "ami_name" : "RHEL-9.0*HVM-202?????-x86_64-*-Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_9.1" : {
      "ami_name" : "RHEL-9.1*HVM-202?????-x86_64-*-Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_9.2" : {
      "ami_name" : "RHEL-9.2*HVM-202?????-x86_64-*-Hourly2-GP2",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_9.3" : {
      "ami_name" : "RHEL-9.3*HVM-202?????-x86_64-*-Hourly2-GP?",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rhel_9" : {
      "ami_name" : "RHEL-9.3*HVM-202?????-x86_64-*-Hourly2-GP?",
      "owner" : "309956199498",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rocky_8.5" : {
      "ami_name" : "Rocky-8-ec2-8.5-202*.x86_64",
      "owner" : "792107900819",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rocky_8.6" : {
      "ami_name" : "Rocky-8-ec2-8.6-202*.x86_64",
      "owner" : "792107900819",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rocky_8.7" : {
      "ami_name" : "Rocky-8-EC2-8.7-202*.x86_64",
      "owner" : "792107900819",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rocky_8.8" : {
      "ami_name" : "Rocky-8-EC2-Base-8.8-202*.x86_64",
      "owner" : "792107900819",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rocky_8" : {
      "ami_name" : "Rocky-8-EC2-Base-8.8-202*.x86_64",
      "owner" : "792107900819",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rocky_9.0" : {
      "ami_name" : "Rocky-9-EC2-9.0-202*.x86_64",
      "owner" : "792107900819",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rocky_9.1" : {
      "ami_name" : "Rocky-9-EC2-Base-9.1-202*.x86_64",
      "owner" : "792107900819",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rocky_9.2" : {
      "ami_name" : "Rocky-9-EC2-Base-9.2-202*.x86_64",
      "owner" : "792107900819",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "rocky_9" : {
      "ami_name" : "Rocky-9-EC2-Base-9.2-202*.x86_64",
      "owner" : "792107900819",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "sles_12" : {
      "ami_name" : "suse-sles-12-sp?-v????????-hvm-ssd-x86_64",
      "owner" : "013907871322",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "sles_12_sp4" : {
      "ami_name" : "suse-sles-12-sp4-v????????-hvm-ssd-x86_64",
      "owner" : "013907871322",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "sles_12_sp5" : {
      "ami_name" : "suse-sles-12-sp5-v????????-hvm-ssd-x86_64",
      "owner" : "013907871322",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "sles_12sp5" : {
      "ami_name" : "suse-sles-12-sp5-v????????-hvm-ssd-x86_64",
      "owner" : "013907871322",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "sles_15" : {
      "ami_name" : "suse-sles-15-sp?-v20??????-hvm-ssd-x86_64",
      "owner" : "013907871322",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "sles_15_sp1" : {
      "ami_name" : "suse-sles-15-sp1-v20??????-hvm-ssd-x86_64",
      "owner" : "013907871322",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "sles_15_sp2" : {
      "ami_name" : "suse-sles-15-sp2-v20??????-hvm-ssd-x86_64",
      "owner" : "013907871322",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "sles_15sp2" : {
      "ami_name" : "suse-sles-15-sp2-v20??????-hvm-ssd-x86_64",
      "owner" : "013907871322",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "sles_15_sp3" : {
      "ami_name" : "suse-sles-15-sp3-v20??????-hvm-ssd-x86_64",
      "owner" : "013907871322",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "sles_15sp3" : {
      "ami_name" : "suse-sles-15-sp3-v20??????-hvm-ssd-x86_64",
      "owner" : "013907871322",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "sles_15sp4" : {
      "ami_name" : "suse-sles-15-sp4-v20??????-hvm-ssd-x86_64",
      "owner" : "013907871322",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "sles_15sp5" : {
      "ami_name" : "suse-sles-15-sp5-v20??????-hvm-ssd-x86_64",
      "owner" : "013907871322",
      "interface" : "eth0"
      "connection" : "ssh",
      "ssh_user" : "ec2-user",
      "ssh_port" : 22
    },
    "ubuntu_16.04" : {
      "ami_name" : "ubuntu/images/hvm-ssd/ubuntu-xenial-16.04-amd64-server-*",
      "owner" : "099720109477",
      "interface" : "ens5"
      "connection" : "ssh",
      "ssh_user" : "ubuntu",
      "ssh_port" : 22
    },
    "ubuntu_18.04" : {
      "ami_name" : "ubuntu/images/hvm-ssd/ubuntu-bionic-18.04-amd64-server-*",
      "owner" : "099720109477",
      "interface" : "ens5"
      "connection" : "ssh",
      "ssh_user" : "ubuntu",
      "ssh_port" : 22
    },
    "ubuntu_20.04" : {
      "ami_name" : "ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*",
      "owner" : "099720109477",
      "interface" : "ens5"
      "connection" : "ssh",
      "ssh_user" : "ubuntu",
      "ssh_port" : 22
    },
    "ubuntu_22.04" : {
      "ami_name" : "ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*",
      "owner" : "099720109477",
      "interface" : "ens5"
      "connection" : "ssh",
      "ssh_user" : "ubuntu",
      "ssh_port" : 22
    },
    "windows_2019" : {
      "ami_name" : "Windows_Server-2019-English-Core-Base-*",
      "owner" : "801119661308",
      "connection" : "ssh",
      "connection" : "winrm",
      "winrm_user" : "Administrator",
      "winrm_useHTTPS" : true
      "winrm_insecure" : true
    },
    "windows_core_2019" : {
      "ami_name" : "Windows_Server-2019-English-Core-Base-*",
      "owner" : "801119661308",
      "interface" : "Ethernet 3"
      "connection" : "winrm",
      "winrm_user" : "Administrator",
      "winrm_useHTTPS" : true
      "winrm_insecure" : true
    },
    "windows_full_2019" : {
      "ami_name" : "Windows_Server-2019-English-Full-Base-*",
      "owner" : "801119661308",
      "interface" : "Ethernet 3"
      "connection" : "winrm",
      "winrm_user" : "Administrator",
      "winrm_useHTTPS" : true
      "winrm_insecure" : true
    },
    "windows_2022" : {
      "ami_name" : "Windows_Server-2022-English-Core-Base-*",
      "owner" : "801119661308",
      "interface" : "Ethernet 3"
      "connection" : "winrm",
      "winrm_user" : "Administrator",
      "winrm_useHTTPS" : true
      "winrm_insecure" : true
    },
    "windows_core_2022" : {
      "ami_name" : "Windows_Server-2022-English-Core-Base-*",
      "owner" : "801119661308",
      "interface" : "Ethernet 3"
      "connection" : "winrm",
      "winrm_user" : "Administrator",
      "winrm_useHTTPS" : true
      "winrm_insecure" : true
    },
    "windows_full_2022" : {
      "ami_name" : "Windows_Server-2022-English-Full-Base-*",
      "owner" : "801119661308",
      "interface" : "Ethernet 3"
      "connection" : "winrm",
      "winrm_user" : "Administrator",
      "winrm_useHTTPS" : true
      "winrm_insecure" : true
    }
  }
}
