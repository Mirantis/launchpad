pipeline {
    agent {
        node {
            label 'amd64 && ubuntu-1804 && overlay2'
        }
    }

    stages {
        stage("Build") {
            steps {
                echo "hello world! maybe now?!?"
                sh "make build"
            }
        }
    }

}
