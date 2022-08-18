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
              image: msr.ci.mirantis.com/prodeng/ci-workspace:stable
              imagePullPolicy: Always
              tty: true
              resources:
                limits: 
                  memory: "32Gi"
                requests:
                  memory: "2Gi"
                  cpu: 2
              volumeMounts:
                - name: docker-socket
                  mountPath: /var/run
            - name: docker-daemon
              image: docker:dind
              imagePullPolicy: Always
              resources:
                limits: 
                  memory: "32Gi"
                requests:
                  memory: "8Gi"
                  cpu: 4
              securityContext:
                privileged: true
              volumeMounts:
                - name: docker-socket
                  mountPath: /var/run
          imagePullSecrets:
            - name: "regcred-msr"
          volumes:
            - name: docker-socket
              emptyDir: {}
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

    stage("Smoke test") {
      parallel {
        stage("'Register' subcommand") {
          stages {
            stage("Register") {
              steps {
                sh "make smoke-register-test"
              }
            }
          }
        }
        stage("Ubuntu 18.04: apply & prune") {
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
      }
    }
  }
}
