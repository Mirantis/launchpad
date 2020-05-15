hosts:
  - address: "127.0.0.1"
    sshPort: 9022
    sshKeyPath: "./id_rsa_mcc"
    user: "root"
    role: "controller"
  - address: "127.0.0.1"
    sshPort: 9023
    sshKeyPath: "./id_rsa_mcc"
    user: "root"
    role: "worker"
ucp:
  installArgs:
    - --admin-username=admin
    - --admin-password=orcaorcaorca
    - --default-node-orchestrator=kubernetes
