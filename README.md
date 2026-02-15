# golang-k8s-microservices

## CI/CD baseline added

This repository now includes:

- `Jenkinsfile` for CI/CD (test, Docker build, optional Docker push)
- `argocd/project.yaml` Argo CD `AppProject`
- `argocd/root-application.yaml` app-of-apps entrypoint
- `argocd/apps/message-service.yaml` Argo CD app for `message-service/k8s`
- `argocd/apps/payment-service.yaml` Argo CD app for `payment-service-deployment/section-hpa-k8s`

## Jenkins setup

1. Create a Jenkins Pipeline job that points to this repository and branch.
2. Jenkins will use the root `Jenkinsfile`.
3. Optional for image push:
   - Add Jenkins credentials (type: username/password).
   - Set credential ID in env var `REGISTRY_CREDENTIALS_ID`.
   - Set Docker registry host in env var `DOCKER_REGISTRY` (example: `docker.io/your-user`).

Pipeline behavior:

- `Unit Test`: runs `go test ./...` for all Go services with `go.mod`
- `Docker Build`: builds images for `inventory-service`, `invoice-service`, `message-service`, `payment-service`
- `Docker Push`: runs only when both `DOCKER_REGISTRY` and `REGISTRY_CREDENTIALS_ID` are set

## Argo CD setup

1. Replace `repoURL` in:
   - `argocd/root-application.yaml`
   - `argocd/apps/message-service.yaml`
   - `argocd/apps/payment-service.yaml`
2. Apply the Argo CD resources:

```bash
kubectl apply -f argocd/project.yaml
kubectl apply -f argocd/root-application.yaml
```

Argo CD will then auto-sync applications under `argocd/apps/`.
