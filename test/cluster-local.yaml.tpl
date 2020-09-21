apiVersion: launchpad.mirantis.com/v1
kind: DockerEnterprise
metadata:
  name: $CLUSTER_NAME
spec:
  hosts:
    - address: "127.0.0.1"
      localhost: true
      role: "manager"
    - address: "$WORKER_IP"
      ssh:
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

