pipeline {
  agent none
  stages {
    stage("Sanity check") {
      parallel {
        stage("Lint") {
          agent any
          steps { sh "make lint" }
        }
        stage("Unit test") {
          agent any
          steps { sh "make unit-test" }
        }
        stage("Build: windows") {
          agent any
          steps { sh "make build GOOS=windows" }
        }
        stage("Build: linux") {
          agent any
          steps { sh "make build GOOS=linux" }
        }
        stage("Build: darwin") {
          agent any
          steps { sh "make build GOOS=darwin" }
        }
      }
    }
    stage("Smoke test") {
      parallel {
        stage("Ubuntu 18.04 with DTR") {
          agent { node { label 'amd64 && ubuntu-1804 && overlay2 && big' } }
          stages {
            stage("Ubuntu 18.04 with DTR: apply") {
              environment {
                UCP_VERSION = "3.2.8"
                DTR_VERSION = "2.7.7"
                LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
                FOOTLOOSE_TEMPLATE = "footloose-dtr.yaml.tpl"
                CONFIG_TEMPLATE = "cluster-dtr.yaml.tpl"
                PRESERVE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-test"
              }
            }
            stage("Ubuntu 18.04: upgrade UCP 3.2 -> 3.3, DTR 2.7 -> 2.8") {
              environment {
                UCP_UPGRADE_VERSION = "3.3.2"
                DTR_UPGRADE_VERSION = "2.8.1"
                LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
                FOOTLOOSE_TEMPLATE = "footloose-dtr.yaml.tpl"
                CONFIG_TEMPLATE = "cluster-dtr.yaml.tpl"
                PRESERVE_CLUSTER = "true"
                REUSE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-upgrade-test"
              }
            }
            stage("Ubuntu 18.04 with DTR: prune") {
              environment {
                LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
                FOOTLOOSE_TEMPLATE = "footloose-dtr.yaml.tpl"
                CONFIG_TEMPLATE = "cluster-dtr.yaml.tpl"
                REUSE_CLUSTER = "true"
                PRESERVE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-prune-test"
              }
            }
            stage("Ubuntu 18.04 with DTR: reset") {
              environment {
                LINUX_IMAGE = "quay.io/footloose/ubuntu18.04"
                FOOTLOOSE_TEMPLATE = "footloose-dtr.yaml.tpl"
                CONFIG_TEMPLATE = "cluster-dtr.yaml.tpl"
                REUSE_CLUSTER = "true"
              }
              steps {
                sh "make smoke-reset-test"
              }
            }
          }
        }
        stage("Ubuntu 18.04: apply catfish") {
          agent { node { label 'amd64 && ubuntu-1804 && overlay2 && big' } }
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
        stage("Ubuntu 16.04") {
          agent { node { label 'amd64 && ubuntu-1804 && overlay2' } }
          stages {
            stage("Ubuntu 16.04: apply") {
              steps {
                sh "make smoke-test LINUX_IMAGE=quay.io/footloose/ubuntu16.04 PRESERVE_CLUSTER=true"
              }
            }
            stage("Ubuntu 16.04: reset") {
              steps {
                sh "make smoke-reset-test LINUX_IMAGE=quay.io/footloose/ubuntu16.04 REUSE_CLUSTER=true"
              }
            }
          }
        }
        stage("CentOS 7: apply") {
          agent { node { label 'amd64 && ubuntu-1804 && overlay2' } }
          steps {
            sh "make smoke-test LINUX_IMAGE=quay.io/footloose/centos7"
          }
        }
        stage("CentOS 8") {
          agent { node { label 'amd64 && ubuntu-1804 && overlay2' } }
          stages {
            stage("CentOS 8: apply") {
              steps {
                sh "make smoke-test PRESERVE_CLUSTER=true LINUX_IMAGE=docker.io/jakolehm/footloose-centos8"
              }
            }
            stage("CentOS 8: reset") {
              steps {
                sh "make smoke-reset-test REUSE_CLUSTER=true LINUX_IMAGE=docker.io/jakolehm/footloose-centos8"
              }
            }
          }
        }
      }
    }
    stage('Release') {
      agent any
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
