cluster:
  name: pusher
  privateKey: ./id_rsa_launchpad
machines:
- count: 1
  backend: docker
  spec:
    image: $LINUX_IMAGE
    name: pusher%d
    privileged: true
    volumes:
    - type: bind
      source: /lib/modules
      destination: /lib/modules
    - type: volume
      destination: /var/lib/containerd
    - type: volume
      destination: /var/lib/docker
    - type: volume
      destination: /var/lib/kubelet
    networks:
    - footloose-cluster
    portMappings:
    - containerPort: 22
      hostPort: 9122
