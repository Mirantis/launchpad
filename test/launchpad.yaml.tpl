apiVersion: launchpad.mirantis.com/v1beta4
kind: DockerEnterprise
metadata:
  name: $CLUSTER_NAME
spec:
  hosts:
    - address: "127.0.0.1"
      ssh:
        port: 9022
        keyPath: "./id_rsa_launchpad"
        user: "root"
      role: "manager"
    - address: "127.0.0.1"
      ssh:
        port: 9023
        keyPath: "./id_rsa_launchpad"
        user: "root"
      role: "worker"
  ucp:
    version: $UCP_VERSION
    imageRepo: $UCP_IMAGE_REPO
    configData: |-
      [scheduling_configuration]
        default_node_orchestrator = "kubernetes"
    installFlags:
      - --admin-username=admin
      - --admin-password=orcaorcaorca
      - --san $UCP_MANAGER_IP
  engine:
    version: $ENGINE_VERSION
