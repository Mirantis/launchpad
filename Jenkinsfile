launchpad_creds = [
  usernamePassword(
    usernameVariable: 'GITHUB_USERNAME',
    passwordVariable: 'GITHUB_TOKEN',
    credentialsId   : 'tools-github-up',
  ),
  usernamePassword(
    usernameVariable: 'REGISTRY_USERNAME',
    passwordVariable: 'REGISTRY_PASSWORD',
    credentialsId   : 'tools-dockerhub-up',
  ),
  string(credentialsId: 'common-digicert--api-key--secret-text', variable: 'SM_API_KEY'),
  file(credentialsId: 'common-digicert--auth-pkcs12--file', variable: 'SM_CLIENT_CERT_FILE'),
  string(credentialsId: 'common-digicert--auth-pkcs12-passphrase--secret-text', variable: 'SM_CLIENT_CERT_PASSWORD'),
]

pipeline {
  agent none
  parameters {
    string(
        defaultValue: 'v1.5.3', 
        name: 'TAG_NAME', 
        trim: true
    )
  }

  stages {
    stage('Release') {
      agent {
        label "linux && pod"
      }
      steps {
        withCredentials(launchpad_creds) {
          sh (
            label: "Executing 'make release'",
            script: """
              make release
            """
          )
        }
      }
    }
  }
}
