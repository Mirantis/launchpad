release_creds = [
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
]

pipeline {
  parameters {
    string(
        name: 'TAG_NAME',
        trim: true
    )
  }
  agent {
    kubernetes {
      yaml """\
apiVersion: v1
kind: Pod
spec:
  imagePullSecrets:
  - name: regcred-registry-mirantis-com
  containers:
  - name: jnlp
    image: registry.mirantis.com/prodeng/ci-workspace:stable
    imagePullPolicy: Always
    resources:
      requests:
        cpu: "0.5"
        memory: 128Mi
  - name: goreleaser
    image: goreleaser/goreleaser:latest
    imagePullPolicy: Always
    resources:
      limits:
        cpu: "4"
      requests:
        cpu: "4"
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
""".stripIndent()
    }
  }

  stages {
    stage('Release') {
      steps {
        container("goreleaser") {
          withCredentials([
            string(credentialsId: 'common-segmentio--api-token--secret-text', variable: 'SEGMENT_TOKEN'),
          ]) {
            sh (
              label: "build clean release",
              script: """
                git checkout \$(git rev-parse --verify ${params.TAG_NAME})
                GORELEASER_CURRENT_TAG=${params.TAG_NAME}} make build-release
              """
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
              label: "signing release binaries (in digicert container)",
              script: """
                make SIGN=./sign sign-release
              """
            )
          }
        }
        container("jnlp") {
          withCredentials(release_creds) {
            sh (
              label: "creating release",
              script: """
                make create-checksum
                make verify-checksum
                ./release.sh
                ./upload_s3.sh
              """
            )
          }
        }
      }
    }
  }
}
