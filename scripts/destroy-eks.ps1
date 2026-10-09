# scripts/destroy-eks.ps1
# Completely tears down the AWS EKS cluster, worker nodes, and VPC to eliminate hourly AWS costs

Write-Host "=========================================" -ForegroundColor Red
Write-Host "  ORBIT PaaS - AWS EKS Teardown Script   " -ForegroundColor Red
Write-Host "=========================================" -ForegroundColor Red

Write-Host "This will terminate the EKS control plane and EC2 worker nodes." -ForegroundColor Yellow
Write-Host "AWS billing for this cluster will drop to $0.00/hour." -ForegroundColor Yellow

$confirmation = Read-Host "Are you sure you want to delete the 'orbit-eks' cluster? (y/N)"
if ($confirmation -ne 'y' -and $confirmation -ne 'Y') {
    Write-Host "Aborted. Cluster was NOT deleted." -ForegroundColor Cyan
    exit 0
}

Write-Host "Deleting EKS Cluster 'orbit-eks' via eksctl..." -ForegroundColor Yellow
eksctl delete cluster -f infra/eksctl-cluster.yaml --wait

if ($LASTEXITCODE -eq 0) {
    Write-Host "EKS Cluster and associated resources successfully deleted. Cost stopped!" -ForegroundColor Green
} else {
    Write-Error "eksctl cluster deletion encountered an error. Please verify in AWS Console."
}
