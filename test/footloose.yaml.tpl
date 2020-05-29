cluster:
  name: ucp
  privateKey: ./id_rsa_launchpad
machines:
- count: 1
  backend: docker
  spec:
    image: $LINUX_IMAGE
    name: manager%d
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
    portMappings:
    - containerPort: 22
      hostPort: 9022
    - containerPort: 443
      hostPort: 443
    - containerPort: 6443
      hostPort: 6443
- count: 1
  backend: docker
  spec:
    image: $LINUX_IMAGE
    name: worker%d
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
    portMappings:
    - containerPort: 22
      hostPort: 9022
