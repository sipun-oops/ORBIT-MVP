# ORBIT PaaS Operations & Shutdown Guide

This document details the complete end-to-end workflow for running the ORBIT PaaS Engine, deploying test applications, and instructions for shutting down the entire environment cleanly.

---

## Part 1: How to Implement and Run (Next Time)

Whenever you want to start fresh or resume work, follow these steps in order:

### 1. Start the Kubernetes Cluster
If you are using a local `kind` cluster, create/start it:
```powershell
# Create the kind cluster
kind create cluster --name orbit
```

### 2. Start the ORBIT Server
Start the core control plane that hosts the dashboard and orchestration logic.
```powershell
cd d:\ORBIT-MVP
go build -o bin/orbit-server.exe .
.\bin\orbit-server.exe
```

### 3. Prepare the Test Application
Ensure your test app (e.g., a simple Flask app with a Dockerfile) is in its own repository (e.g., `orbit-test-app`).
If you need to push updates:
```powershell
cd d:\orbit-test-app
git add .
git commit -m "Update app"
git push -u origin main
```

### 4. Trigger a Deployment
1. Open your browser and navigate to the local dashboard: `http://localhost:8080/`
2. Provide the GitHub repository URL of your test application (e.g., `https://github.com/sipun-oops/orbit-test-app.git`).
3. Click deploy. The engine will pull the code, build the mock/actual docker image, and apply it to the Kubernetes cluster.

### 5. Access the Deployed Application
Since `kind` does not natively support `LoadBalancer` IPs, use port-forwarding to access your app:
```powershell
# In a new PowerShell window
kubectl port-forward -n orbit-apps svc/flask-simple-test 5056:8080
```
Then visit `http://localhost:5056` in your browser.

---

## Part 2: Complete Shutdown Instructions (Do This Now)

To completely shut down the environment, clean up resources, and prevent background tasks from draining memory/CPU, run the following steps:

### 1. Stop the ORBIT Server
If the server `.\bin\orbit-server.exe` is running in your terminal, press `Ctrl + C` to stop it.

### 2. Stop Port-Forwarding
If you have any `kubectl port-forward` commands running in the background or other terminals, press `Ctrl + C` in those terminals to stop them.

### 3. Delete the Deployed Application
If you want to keep the cluster but remove the app:
```powershell
kubectl delete namespace orbit-apps
```

### 4. Delete the Local Kubernetes Cluster
To completely wipe the local Kubernetes environment (this will delete the `orbit` kind cluster and all associated containers):
```powershell
kind delete cluster --name orbit
```
*Note: This is the most effective way to guarantee all Kubernetes resources, pods, and simulated load balancers are stopped.*

### 5. Prune Unused Docker Images (Optional)
During testing, many Docker images can accumulate. To free up space:
```powershell
docker system prune -f
```

---

## Part 3: AWS Services Shutdown (If Applicable)

If you have provisioned actual AWS resources (EKS, ECR, CodeBuild) instead of using the local mock mode, you must clean them up to prevent ongoing hourly billing.

### 1. Destroy the EKS Cluster
The repository includes a dedicated script to completely tear down the AWS EKS cluster, worker nodes, and associated VPC resources.
```powershell
cd d:\ORBIT-MVP
.\scripts\destroy-eks.ps1
```
*This script uses `eksctl delete cluster` under the hood. It takes a few minutes but drops billing to $0.00/hour for compute.*

### 2. Clean up ECR Repositories
Docker images stored in Elastic Container Registry (ECR) incur storage costs. If you want to delete them:
1. Go to the **Amazon ECR Console**.
2. Find the repositories created by the ORBIT engine (e.g., `orbit/flask-simple-test`).
3. Select them and click **Delete**.

### 3. Clean up CodeBuild Projects
CodeBuild projects generally do not cost money when they are not running, but to keep your AWS account clean:
1. Go to the **AWS CodeBuild Console**.
2. Select the projects created by ORBIT.
3. Click **Delete**.
