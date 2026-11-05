pipeline {
  parameters {
    string(
        name: 'TAG_NAME',
        trim: true
    )
  }

  environment {
    // TAG_NAME reaches the shell through the environment and is never spliced
    // into a Groovy string. Validated by the 'Validate tag' stage below.
    TAG_NAME = params.TAG_NAME
  }
  agent {
    kubernetes {
      yaml """\
apiVersion: v1
kind: Pod
metadata:
  annotations:
    cluster-autoscaler.kubernetes.io/safe-to-evict: false
spec:
  imagePullSecrets:
  - name: regcred-registry-mirantis-com
  containers:
  - name: jnlp
    image: "registry.mirantis.com/prodeng/ci-workspace:1.0.8"
    securityContext:
     runAsUser: 0
     runAsGroup: 0
    imagePullPolicy: Always
    volumeMounts:
    - name: docker-socket
      mountPath: /var/run
    - name: tmp-dir
      mountPath: /tmp
    resources:
      limits:
        cpu: "1"
      requests:
        cpu: "0.5"
  - name: docker-daemon
    image: "docker:24.0.9-dind"
    securityContext:
     privileged: true
    volumeMounts:
     - name: docker-socket
       mountPath: /var/run
     - name: tmp-dir
       mountPath: /tmp
    resources:
      limits:
        cpu: 1
      requests:
        cpu: 0.5
  - name: goreleaser
    image: goreleaser/goreleaser:latest
    imagePullPolicy: Always
    resources:
      limits:
        cpu: 4
      requests:
        cpu: 4
    command:
    - sleep
    args:
    - 99d
  - name: digicert
    image: registry.mirantis.com/prodeng/digicert-keytools-jsign:latest
    imagePullPolicy: Always
    resources:
      requests:
        cpu: "1"
        memory: 4Gi
    command:
    - sleep
    args:
    - 99d
  volumes:
  - name: docker-socket
    emptyDir: {}
  - name: tmp-dir # This volume mount is required since SSH_AUTH_SOCK is created in /tmp
    emptyDir: {}
""".stripIndent()
    }
  }

  stages {
    stage('Validate tag') {
      steps {
        script {
          // Single-quoted Groovy string, so nothing here is interpolated and
          // there is no slashy-string '$/' ambiguity. ==~ is a full match, so
          // a trailing newline cannot smuggle a second line past the anchors.
          if (!((params.TAG_NAME ?: '') ==~ '^v[0-9][0-9A-Za-z.\\-+]*$')) {
            // The rejected value is deliberately not echoed: it is attacker
            // controlled, and interpolating it here would reintroduce the exact
            // Groovy interpolation this pipeline must not contain.
            error 'TAG_NAME must match ^v[0-9][0-9A-Za-z.\\-+]*$ (e.g. v1.5.19, v1.5.19-tp2)'
          }
        }
      }
    }
    stage('Release') {
      steps {
        container("goreleaser") {
          withCredentials([
            string(credentialsId: 'tools-segment--launchpad-production-token--secret-text', variable: 'SEGMENT_TOKEN'),
          ]) {
            // Single-quoted script: Groovy passes it through verbatim and the
            // shell dereferences $TAG_NAME from the environment. SEGMENT_TOKEN
            // is already exported into this step by withCredentials above, so
            // make inherits it without it appearing in the script text or argv.
            sh (
              label: 'build clean release',
              script: '''
                git checkout "$(git rev-parse --verify "$TAG_NAME")"
                GORELEASER_CURRENT_TAG="$TAG_NAME" make build-release
              '''
            )
          }
        }
        container("digicert") {
          withCredentials([
            string(credentialsId: 'common-digicert--api-key--secret-text', variable: 'SM_API_KEY'),
            file(credentialsId: 'common-digicert--auth-pkcs12--file', variable: 'SM_CLIENT_CERT_FILE'),
            string(credentialsId: 'common-digicert--auth-pkcs12-passphrase--secret-text', variable: 'SM_CLIENT_CERT_PASSWORD'),
          ]) {
            sh (
              label: 'signing release binaries (in digicert container)',
              script: '''
                make SIGN=./sign sign-release
              '''
            )
          }
        }
        container("jnlp") {
          withCredentials([
            [
              $class: 'AmazonWebServicesCredentialsBinding',
              accessKeyVariable: 'AWS_ACCESS_KEY_ID',
              secretKeyVariable: 'AWS_SECRET_ACCESS_KEY',
              credentialsId: 'tools-aws-de-production-access-keys',
            ],
            usernamePassword(
              usernameVariable: 'GITHUB_USERNAME',
              passwordVariable: 'GITHUB_TOKEN',
              credentialsId   : 'tools-github-up',
            ),
          ]) {
            sh (
              label: 'creating release',
              script: '''
                make create-checksum
                make verify-checksum
                ./release.sh
                ./upload_s3.sh
              '''
            )
          }
        }
      }
    }
  }
}
