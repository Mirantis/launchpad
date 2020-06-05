apiVersion: launchpad.mirantis.com/v1beta2
kind: UCP
metadata:
  name: $CLUSTER_NAME
spec:
  hosts:
    - address: "127.0.0.1"
      sshPort: 9022
      sshKeyPath: "./id_rsa_launchpad"
      user: "root"
      role: "manager"
    - address: "127.0.0.1"
      sshPort: 9023
      sshKeyPath: "./id_rsa_launchpad"
      user: "root"
      role: "worker"
  ucp:
    version: $UCP_VERSION
    configData: |-
      [scheduling_configuration]
        default_node_orchestrator = "kubernetes"
    installFlags:
      - --admin-username=admin
      - --admin-password=orcaorcaorca
  engine:
    version: $ENGINE_VERSION
