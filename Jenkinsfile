def docker_hub = [
  usernamePassword(
    usernameVariable: 'REGISTRY_USERNAME',
    passwordVariable: 'REGISTRY_PASSWORD',
    credentialsId   : 'docker-hub-generic-up',
  )
]

pipeline {
  agent {
    node {
      label 'amd64 && ubuntu-1804 && overlay2'
    }
  }

  stages {
    stage("Build") {
      steps {
        sh "make unit-test"
        sh "make lint"
        sh "make build-all"
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
        stage("Ubuntu 20.04: apply & reset") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          stages {
            stage("Apply") {
              environment {
                LINUX_IMAGE = "quay.io/footloose/ubuntu20.04"
                PRESERVE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-apply-test"
              }
            }
            stage("Reset") {
              environment {
                LINUX_IMAGE = "quay.io/footloose/ubuntu20.04"
                REUSE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-reset-test"
              }
            }
          }
        }
        stage("CentOS 7: apply") {
          agent {
              node {
                  label 'amd64 && ubuntu-1804 && overlay2 && big'
              }
          }
          steps {
            sh "make smoke-test LINUX_IMAGE=quay.io/footloose/centos7"
          }
        }
        stage("CentOS 8") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
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
        stage("Ubuntu 16.04 apply") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
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
              label 'amd64 && ubuntu-1804 && overlay2 && big'
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
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          stages {
            stage("Install MKE3.3.7 MSR2.8 MCR19.03.14") {
              environment {
                LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
                FOOTLOOSE_TEMPLATE = "footloose-msr.yaml.tpl"
                LAUNCHPAD_CONFIG = "launchpad-msr.yaml"
                MKE_VERSION = "3.3.7"
                MKE_IMAGE_REPO = "docker.io/mirantis"
                MSR_VERSION = "2.8.5"
                MSR_IMAGE_REPO = "docker.io/mirantis"
                MCR_VERSION = "19.03.14"
                MCR_CHANNEL = "stable"
                MCR_REPO_URL = "https://repos.mirantis.com"
                PRESERVE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-test"
              }
            }
            stage("Upgrade MCR from test channel and prune MSR") {
              environment {
                LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
                FOOTLOOSE_TEMPLATE = "footloose-msr.yaml.tpl"
                LAUNCHPAD_CONFIG = "launchpad-msr-beta.yaml"
                MKE_VERSION = "3.3.7"
                MKE_IMAGE_REPO = "docker.io/mirantis"
                MSR_VERSION = "2.8.5"
                MSR_IMAGE_REPO = "docker.io/mirantis"
                MCR_VERSION = "20.10.0"
                MCR_CHANNEL = "test"
                MCR_REPO_URL = "https://repos.mirantis.com"
                REUSE_CLUSTER = "true"
                PRESERVE_CLUSTER = "true"
              }
              steps {
                withCredentials(docker_hub) {
                  sh "make smoke-test"
                  sh "make smoke-prune-test"
                  sh "make smoke-cleanup"
                }
              }
            }
            stage("Upgrade MKE3.4 beta MSR2.9 beta from private repos and re-add pruned MSR") {
              environment {
                LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
                FOOTLOOSE_TEMPLATE = "footloose-msr.yaml.tpl"
                LAUNCHPAD_CONFIG = "launchpad-msr-beta.yaml"
                MKE_VERSION = "3.4.1-48de8b4"
                MKE_IMAGE_REPO = "docker.io/mirantiseng"
                MSR_IMAGE_REPO = "docker.io/mirantiseng"
                MSR_VERSION = "2.9.0-tp3"
                MCR_VERSION = "20.10.0"
                MCR_CHANNEL = "test"
                MCR_REPO_URL = "https://repos.mirantis.com"
                REUSE_CLUSTER = "true"
              }
              steps {
                withCredentials(docker_hub) {
                  sh "make smoke-test"
                  sh "make smoke-cleanup"
                }
              }
            }
          }
        }
      }
    }
    stage('Release') {
      when {
        buildingTag()
      }
      steps {
        withCredentials([
          string(credentialsId: "00efb2fb-3e89-4d75-b225-f0c37746df54", variable: "GITHUB_TOKEN"),
          string(credentialsId: "launchpad-win-certificate", variable: "WIN_PKCS12"),
          string(credentialsId: "launchpad-win-certificate-passwd", variable: "WIN_PKCS12_PASSWD"),
        ]) {
          sh "make release"
        }
      }
    }
  }
}
