apiVersion: launchpad.mirantis.com/v1
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
    - address: "127.0.0.1"
      ssh:
        port: 9024
        keyPath: "./id_rsa_launchpad"
        user: "root"
      role: "dtr"
    - address: "127.0.0.1" # REMOVE_THIS
      ssh: # REMOVE_THIS
        port: 9025 # REMOVE_THIS
        keyPath: "./id_rsa_launchpad" # REMOVE_THIS
        user: "root" # REMOVE_THIS
      role: "dtr" # REMOVE_THIS
  cluster:
    prune: false
  ucp:
    version: $UCP_VERSION
    imageRepo: $UCP_IMAGE_REPO
    configData: |-
      [scheduling_configuration]
        default_node_orchestrator = "kubernetes"
        enable_admin_ucp_scheduling = true
    installFlags:
      - --admin-username=admin
      - --admin-password=orcaorcaorca
  engine:
    version: $ENGINE_VERSION
  dtr:
    version: $DTR_VERSION
    imageRepo: $DTR_IMAGE_REPO
    installFlags:
      - --ucp-url $UCP_MANAGER_IP
      - --ucp-insecure-tls
      - --replica-http-port 81
      - --replica-https-port 444
    replicaConfig: sequential
