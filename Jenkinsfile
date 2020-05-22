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
        stage("Ubuntu 18.04") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2'
            }
          }
          steps {
            sh "make smoke-test LINUX_IMAGE=quay.io/footloose/ubuntu18.04"
          }
        }
        stage("CentOS 7") {
          agent {
              node {
                  label 'amd64 && ubuntu-1804 && overlay2'
              }
          }
          steps {
            sh "make smoke-test LINUX_IMAGE=quay.io/footloose/centos7"
          }
        }
        stage("CentOS 8") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2'
            }
          }
          steps {
            sh "make smoke-test LINUX_IMAGE=docker.io/jakolehm/footloose-centos8"
          }
        }
      }
    }
    stage("Smoke test upgrade") {
      parallel {
        stage("Upgrade 3.2 -> 3.3 (Ubuntu 18.04)") {
          agent {
            node {
              label 'amd64 && ubuntu-1804 && overlay2'
            }
          }
          steps {
            sh "make smoke-upgrade-test LINUX_IMAGE=quay.io/footloose/ubuntu18.04"
          }
        }
      }
    }
    stage('Release') {
      when {
        buildingTag()
      }
      steps {
        withCredentials([string(credentialsId: "00efb2fb-3e89-4d75-b225-f0c37746df54", variable: "GITHUB_TOKEN")]) {
          sh "make release"
        }
      }
    }
  }
}
