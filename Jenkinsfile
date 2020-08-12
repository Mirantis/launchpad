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
        stage("Ubuntu 16.04: apply") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          steps {
            sh "make smoke-test LINUX_IMAGE=quay.io/footloose/ubuntu16.04"
          }
        }
        stage("Ubuntu 18.04: apply") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          steps {
            sh "make smoke-test LINUX_IMAGE=quay.io/footloose/ubuntu18.04"
          }
        }
        stage("Ubuntu 18.04 with DTR: apply") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          environment {
            LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
            FOOTLOOSE_TEMPLATE = "footloose-dtr.yaml.tpl"
            CONFIG_TEMPLATE = "cluster-dtr.yaml.tpl"
          }
          steps {
            sh "make smoke-test"
          }
        }
        stage("Ubuntu 18.04: apply v1beta1") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          steps {
            sh "make smoke-test CONFIG_TEMPLATE=v1beta1_cluster.yaml.tpl LINUX_IMAGE=quay.io/footloose/ubuntu18.04"
          }
        }
        stage("Ubuntu 18.04: apply catfish") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          environment {
            UCP_IMAGE_REPO = "docker.io/dockereng"
            UCP_VERSION = "3.4.0-tp6"
            ENGINE_VERSION = "19.03.11"
            REGISTRY_CREDS = credentials("dockerbuildbot-index.docker.io")
          }
          steps {
            sh "make smoke-test LINUX_IMAGE=quay.io/footloose/ubuntu18.04"
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
        stage("CentOS 8: apply") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          steps {
            sh "make smoke-test LINUX_IMAGE=docker.io/jakolehm/footloose-centos8"
          }
        }
        stage("Ubuntu 18.04: upgrade UCP 3.2 -> 3.3, DTR 2.7 -> 2.8") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          environment {
            LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
            FOOTLOOSE_TEMPLATE = "footloose-dtr.yaml.tpl"
            CONFIG_TEMPLATE = "cluster-dtr.yaml.tpl"
          }
          steps {
            sh "make smoke-upgrade-test"
          }
        }
        stage("Ubuntu 18.04 with DTR: prune") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          environment {
            LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
            FOOTLOOSE_TEMPLATE = "footloose-dtr.yaml.tpl"
            CONFIG_TEMPLATE = "cluster-dtr.yaml.tpl"
          }
          steps {
            sh "make smoke-prune-test"
          }
        }
        stage("Ubuntu 18.04: with docker credentials") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          environment {
            LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
            FOOTLOOSE_TEMPLATE = "footloose-dtr.yaml.tpl"
            CONFIG_TEMPLATE = "cluster-dtr.yaml.tpl"
            UCP_IMAGE_REPO = "docker.io/dockereng"
            REGISTRY_CREDS = credentials("dockerbuildbot-index.docker.io")
          }
          steps {
            sh "make smoke-test"
          }
        }
        stage("Ubuntu 18.04: reset") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          steps {
            sh "make smoke-reset-test LINUX_IMAGE=quay.io/footloose/ubuntu18.04"
          }
        }
        stage("Ubuntu 18.04 with DTR: reset") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          environment {
            LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
            FOOTLOOSE_TEMPLATE = "footloose-dtr.yaml.tpl"
            CONFIG_TEMPLATE = "cluster-dtr.yaml.tpl"
          }
          steps {
            sh "make smoke-reset-test"
          }
        }
        stage("Ubuntu 16.04: reset") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          steps {
            sh "make smoke-reset-test LINUX_IMAGE=quay.io/footloose/ubuntu16.04"
          }
        }
        stage("CentOS 8: reset") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2 && big'
            }
          }
          steps {
            sh "make smoke-reset-test LINUX_IMAGE=docker.io/jakolehm/footloose-centos8"
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
