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
            - name: builder
              image: golang:1.19
              command:
                - cat
              tty: true
            - name: jnlp
              image: docker:latest
              command:
                - apk-add --update alpine-sdk
                - cat
              tty: true
              volumeMounts:
                - name: docker-socket
                  mountPath: /var/run
            - name: docker-daemon
              image: docker:dind
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
        container("builder") {
          sh "make unit-test"
          sh "make lint || echo 'Lint failed'"
          sh "make build-all"
        }
      }
    }

    stage("Smoke test") {
      parallel {
        stage("'Register' subcommand") {
          stages {
            stage("Register") {
              steps {
                container("runner") {
                  sh "make bin/launchpad"
                  sh "make smoke-register-test"
                }
              }
            }
          }
        }
      }
    }
  }
}
