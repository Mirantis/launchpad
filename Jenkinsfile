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
        stage("Ubuntu 18.04: apply & reset") {
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
                sh "make smoke-apply-test LINUX_IMAGE=quay.io/footloose/ubuntu18.04"
              }
            }
            stage("Reset") {
              environment {
                REUSE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-reset-test LINUX_IMAGE=quay.io/footloose/ubuntu18.04"
              }
            }
          }
        }
        stage("Ubuntu 18.04: apply v1beta1") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          steps {
            sh "make smoke-apply-test LAUNCHPAD_CONFIG=launchpad-v1beta1.yaml LINUX_IMAGE=quay.io/footloose/ubuntu18.04"
          }
        }
        stage("Ubuntu 18.04: upload images") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          steps {
            sh "make smoke-apply-upload-test CONFIG_TEMPLATE=launchpad-upload.yaml.tpl LINUX_IMAGE=quay.io/footloose/ubuntu18.04"
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
        stage("Ubuntu 16.04") {
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
                sh "make smoke-apply-test LINUX_IMAGE=quay.io/footloose/ubuntu16.04"
              }
            }
            stage("Reset") {
              environment {
                REUSE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-reset-test LINUX_IMAGE=quay.io/footloose/ubuntu16.04"
              }
            }
          }
        }
        stage("UCP3.3.3 VXLAN switch") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          stages {
            stage("VXLAN:false") {
              environment {
                LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
                LAUNCHPAD_CONFIG = "launchpad-vxlan.yaml"
                CALICO_VXLAN = "false"
                PRESERVE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-test"
              }
            }
            stage("VXLAN:true") {
              environment {
                LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
                LAUNCHPAD_CONFIG = "launchpad-vxlan.yaml"
                CALICO_VXLAN = "true"
                REUSE_CLUSTER = "true"
                MUST_FAIL = "true"
              }
              steps {
                sh "make smoke-test"
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
        stage("Ubuntu 18.04 with DTR") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          stages {
            stage("Install UCP3.2 DTR2.7 ENG19.03.8") {
              environment {
                LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
                FOOTLOOSE_TEMPLATE = "footloose-dtr.yaml.tpl"
                LAUNCHPAD_CONFIG = "launchpad-dtr.yaml"
                UCP_VERSION = "3.2.8"
                IMAGE_REPO = "docker.io/mirantis"
                DTR_VERSION = "2.7.8"
                DTR_IMAGE_REPO = "docker.io/mirantis"
                ENGINE_VERSION = "19.03.8"
                PRESERVE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-test"
              }
            }
            stage("Upgrade UCP3.3 DTR2.8 ENG19.03.12") {
              environment {
                LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
                FOOTLOOSE_TEMPLATE = "footloose-dtr.yaml.tpl"
                LAUNCHPAD_CONFIG = "launchpad-dtr.yaml"
                UCP_VERSION = "3.3.3"
                REGISTRY_CREDS = credentials("dockerbuildbot-index.docker.io")
                IMAGE_REPO = "docker.io/mirantis"
                DTR_VERSION = "2.8.3"
                ENGINE_VERSION = "19.03.12"
                REUSE_CLUSTER = "true"
                PRESERVE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-test"
                sh "make smoke-prune-test"
                sh "make smoke-reset-test"
                sh "make smoke-cleanup"
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
