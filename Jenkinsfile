pipeline {
  agent any

  options {
    ansiColor('xterm')
    timestamps()
    disableConcurrentBuilds()
  }

  environment {
    GO_TEST_SERVICES = "inventory-service invoice-service message-service order-service payment-service queue-service"
    DOCKER_SERVICES = "inventory-service invoice-service message-service payment-service"
    IMAGE_TAG = "${env.GIT_COMMIT ? env.GIT_COMMIT.take(7) : env.BUILD_NUMBER}"
    REGISTRY = "${env.DOCKER_REGISTRY ?: ''}"
    REGISTRY_CREDENTIALS_ID = "${env.REGISTRY_CREDENTIALS_ID ?: ''}"
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
      }
    }

    stage('Unit Test') {
      agent {
        docker {
          image 'golang:1.25'
          reuseNode true
        }
      }
      steps {
        sh '''
          set -euo pipefail

          for svc in ${GO_TEST_SERVICES}; do
            if [ -f "${svc}/go.mod" ]; then
              echo "==> go test ./${svc}/..."
              (cd "${svc}" && go test ./...)
            fi
          done
        '''
      }
    }

    stage('Docker Build') {
      steps {
        sh '''
          set -euo pipefail

          for svc in ${DOCKER_SERVICES}; do
            if [ -f "${svc}/Dockerfile" ]; then
              echo "==> docker build ${svc}:${IMAGE_TAG}"
              docker build -t "${svc}:${IMAGE_TAG}" "${svc}"
            fi
          done
        '''
      }
    }

    stage('Docker Push') {
      when {
        expression { return env.REGISTRY?.trim() && env.REGISTRY_CREDENTIALS_ID?.trim() }
      }
      steps {
        withCredentials([usernamePassword(
          credentialsId: "${REGISTRY_CREDENTIALS_ID}",
          usernameVariable: 'REGISTRY_USER',
          passwordVariable: 'REGISTRY_PASS'
        )]) {
          sh '''
            set -euo pipefail
            echo "${REGISTRY_PASS}" | docker login -u "${REGISTRY_USER}" --password-stdin "${REGISTRY}"

            for svc in ${DOCKER_SERVICES}; do
              if [ -f "${svc}/Dockerfile" ]; then
                docker tag "${svc}:${IMAGE_TAG}" "${REGISTRY}/${svc}:${IMAGE_TAG}"
                docker push "${REGISTRY}/${svc}:${IMAGE_TAG}"
              fi
            done
          '''
        }
      }
    }
  }

  post {
    always {
      sh '''
        set +e
        docker logout "${REGISTRY}" >/dev/null 2>&1 || true
      '''
    }
    success {
      echo "Pipeline finished. Image tag: ${IMAGE_TAG}"
    }
  }
}
