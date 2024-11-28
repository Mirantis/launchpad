# Infrastructure provision and install

## Provisioning

- This terraform configuration creates infrastructure required for running Launchpad

## Installation

- This configuration uses Launchpad terraform provider for installing MCR, MKE and MSR 2.9x on the provisioned infrstructure
- After installing MKE, the configuration uses MKE terraform provider to create a kubernetes context to install kubernetes workloads
- After creating a kubernetes context the same is used for installing AWS cloud controller manager and AWS EBS CSI driver
- The content of test.tf file is commented out and the resources in that file can be used testing the functionality of AWS cloud controller manager and AWS EBS CSI driver