# scripts/create-eks.ps1
# Spins up the AWS EKS cluster on-demand using eksctl

Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "  ORBIT PaaS - AWS EKS Cluster Provisioner" -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan

Write-Host "[1/3] Verifying AWS Identity..." -ForegroundColor Yellow
aws sts get-caller-identity
if ($LASTEXITCODE -ne 0) {
    Write-Error "AWS authentication failed. Check credentials."
    exit 1
}

Write-Host "[2/3] Provisioning EKS Cluster 'orbit-eks' in ap-south-1 (~15 mins)..." -ForegroundColor Yellow
eksctl create cluster -f infra/eksctl-cluster.yaml
if ($LASTEXITCODE -ne 0) {
    Write-Error "eksctl cluster creation failed."
    exit 1
}

Write-Host "[3/3] Updating local kubeconfig..." -ForegroundColor Yellow
aws eks update-kubeconfig --region ap-south-1 --name orbit-eks

Write-Host "Cluster nodes:" -ForegroundColor Green
kubectl get nodes

Write-Host "EKS cluster is live and ready!" -ForegroundColor Green
