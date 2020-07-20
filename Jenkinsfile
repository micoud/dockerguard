pipeline {
  agent any

  environment {
    REGISTRY="registry.cta-test.zeuthen.desy.de"
  }

  stages {
    stage('Build container') {
      steps {
        sh "docker build -t ${REGISTRY}/dockerguard:latest ."
      }
    }

    stage('Push container to registry') {
      steps {
        sh "docker push ${REGISTRY}/dockerguard:latest"
      }
    }
  }
}
