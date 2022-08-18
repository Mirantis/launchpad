def docker_hub = [
  usernamePassword(
    usernameVariable: 'REGISTRY_USERNAME',
    passwordVariable: 'REGISTRY_PASSWORD',
    credentialsId   : 'docker-hub-generic-up',
  )
]

pipeline {
  agent {
    kubernetes {
      yaml '''
        apiVersion: v1
        kind: Pod
        spec:
            containers:
            - name: jnlp
              image: golang:1.19
              imagePullPolicy: Always
              volumeMounts:
              - name: docker-socket
                mountPath: /var/run
              resources:
                limits:
                  cpu: 4
                  memory: 16Gi
                requests:
                  cpu: 2
                  memory: 4Gi
            - name: docker-daemon
              image: docker:dind
              securityContext:
               privileged: true
              volumeMounts:
               - name: docker-socket
                 mountPath: /var/run
              resources:
                limits:
                  cpu: 4
                  memory: 16Gi
                requests:
                  cpu: 2
                  memory: 4Gi
            volumes:
            - name: docker-socket
              emptyDir: {}
            tolerations:
            - key: "app"
              operator: "Equal"
              value: "jenkins-agents"
              effect: "NoExecute"
            nodeSelector:
              app: "jenkins-agents"
      '''
    }
  }

  stages {
    stage("Build") {
      steps {
        sh "make unit-test"
        sh "make lint || echo 'Lint failed'"
        sh "make build-all"
      }
    }
    stage('Release') {
      when {
        buildingTag()
      }
      steps {
        withCredentials([
          string(credentialsId: "launchpad-github-up", variable: "GITHUB_TOKEN"),
          /*
          string(credentialsId: "launchpad-win-certificate", variable: "WIN_PKCS12"),
          string(credentialsId: "launchpad-win-certificate-passwd", variable: "WIN_PKCS12_PASSWD"),
          */
        ]) {
          sh "make release"
        }
      }
    }
    stage("Smoke test") {
      parallel {
        stage("Register subcommand") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2'
            }
          }
          stages {
            stage("Register") {
              steps {
                sh "make smoke-register-test"
              }
            }
          }
        }
        stage("Ubuntu 18.04: apply & prune") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2'
            }
          }
          stages {
            stage("Apply") {
              environment {
                LAUNCHPAD_CONFIG = "launchpad-prune.yaml"
                FOOTLOOSE_TEMPLATE = "footloose-prune.yaml.tpl"
                PRESERVE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-apply-test LINUX_IMAGE=quay.io/footloose/ubuntu18.04"
              }
            }
            stage("Prune") {
              environment {
                LAUNCHPAD_CONFIG = "launchpad-prune.yaml"
                FOOTLOOSE_TEMPLATE = "footloose-prune.yaml.tpl"
                REUSE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-prune-test LINUX_IMAGE=quay.io/footloose/ubuntu18.04"
              }
            }
          }
        }
        stage("Ubuntu 18.04: apply with SSH bastion host") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2'
            }
          }
          stages {
            stage("Apply") {
              environment {
                LAUNCHPAD_CONFIG = "launchpad-bastion.yaml"
                FOOTLOOSE_TEMPLATE = "footloose-bastion.yaml.tpl"
              }
              steps {
                sh "make smoke-apply-bastion-test LINUX_IMAGE=quay.io/footloose/ubuntu18.04"
              }
            }
          }
        }
        stage("Ubuntu 18.04: apply with SSH auth forwarding") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2'
            }
          }
          stages {
            stage("Apply") {
              environment {
                LAUNCHPAD_CONFIG = "launchpad-forward.yaml"
                FOOTLOOSE_TEMPLATE = "footloose-bastion.yaml.tpl"
              }
              steps {
                sh "make smoke-apply-forward-test LINUX_IMAGE=quay.io/footloose/ubuntu18.04"
              }
            }
          }
        }
        stage("CentOS 7: apply") {
          agent {
              node {
                  label 'amd64 && ubuntu-1804 && overlay2'
              }
          }
          steps {
            sh "make smoke-test LINUX_IMAGE=quay.io/footloose/centos7"
          }
        }
/*
        stage("CentOS 8") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2'
            }
          }
          stages {
            stage("Apply") {
              environment {
                PRESERVE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-test LINUX_IMAGE=docker.io/jakolehm/footloose-centos8"
              }
            }
            stage("Reset") {
              environment {
                REUSE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-reset-test LINUX_IMAGE=docker.io/jakolehm/footloose-centos8"
              }
            }
          }
        }
*/
        stage("Ubuntu 16.04 apply") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2'
            }
          }
          stages {
            stage("Apply") {
              steps {
                sh "make smoke-apply-test LINUX_IMAGE=quay.io/footloose/ubuntu16.04"
              }
            }
          }
        }
        stage("Ubuntu 18.04: local worker apply") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2'
            }
          }
          environment {
            LAUNCHPAD_CONFIG = "launchpad-local.yaml"
            FOOTLOOSE_TEMPLATE = "footloose-local.yaml.tpl"
          }
          steps {
            sh "make smoke-apply-test-localhost LINUX_IMAGE=quay.io/footloose/ubuntu18.04"
          }
        }
        stage("Ubuntu 18.04 upgrades and MSR") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2'
            }
          }
          stages {
            stage("Install MKE3.3.5 MSR2.7 MCR19.03.8") {
              environment {
                LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
                FOOTLOOSE_TEMPLATE = "footloose-msr.yaml.tpl"
                LAUNCHPAD_CONFIG = "launchpad-msr.yaml"
                MKE_VERSION = "3.3.5"
                MKE_IMAGE_REPO = "docker.io/mirantis"
                MSR_VERSION = "2.7.8"
                MSR_IMAGE_REPO = "docker.io/mirantis"
                MCR_VERSION = "19.03.14"
                PRESERVE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-test"
              }
            }
            stage("Upgrade MCR, MSR & MKE") {
              environment {
                LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
                FOOTLOOSE_TEMPLATE = "footloose-msr.yaml.tpl"
                LAUNCHPAD_CONFIG = "launchpad-msr-beta.yaml"
                MKE_VERSION = "3.3.7"
                MKE_IMAGE_REPO = "docker.io/mirantis"
                MSR_VERSION = "2.8.5"
                MSR_IMAGE_REPO = "docker.io/mirantis"
                MCR_VERSION = "20.10.0"
                REUSE_CLUSTER = "true"
                PRESERVE_CLUSTER = "true"
              }
              steps {
                withCredentials(docker_hub) {
                  sh "make smoke-test"
                }
              }
            }
            stage("Upgrade MKE3.4 beta MSR2.9 beta, MCR20.10 from private repos") {
              environment {
                LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
                FOOTLOOSE_TEMPLATE = "footloose-msr.yaml.tpl"
                LAUNCHPAD_CONFIG = "launchpad-msr-beta.yaml"
                MKE_VERSION = "3.4.6-21f9b26"
                MKE_IMAGE_REPO = "docker.io/mirantiseng"
                MSR_IMAGE_REPO = "docker.io/mirantiseng"
                MSR_VERSION = "2.9.0-tp3"
                MCR_VERSION = "20.10.0"
                REUSE_CLUSTER = "true"
              }
              steps {
                withCredentials(docker_hub) {
                  sh "make smoke-test"
                }
              }
            }
          }
        }
      }
    }
  }
}
