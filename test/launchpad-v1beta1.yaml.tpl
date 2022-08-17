apiVersion: launchpad.mirantis.com/v1beta1
kind: UCP
metadata:
  name: $CLUSTER_NAME
spec:
  hosts:
    - address: "127.0.0.1"
      sshPort: 9022
      sshKeyPath: "./id_rsa_launchpad"
      user: "launchpad"
      role: "manager"
    - address: "127.0.0.1"
      sshPort: 9023
      sshKeyPath: "./id_rsa_launchpad"
      user: "launchpad"
      role: "worker"
  ucp:
    version: $MKE_VERSION
    configData: |-
      [scheduling_configuration]
        default_node_orchestrator = "kubernetes"
    installFlags:
      - --admin-username=admin
      - --admin-password=orcaorcaorca
      - --force-minimums
  engine:
    version: $MCR_VERSION
