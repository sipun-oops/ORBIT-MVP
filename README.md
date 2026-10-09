# ORBIT PaaS Engine (Simplified MVP)

A lightweight, self-hosted Git-to-Kubernetes Platform as a Service (PaaS) engine designed for AWS. 

This MVP demonstrates a complete end-to-end automated cloud application delivery pipeline in just a few files of Go.

## Architecture

This project is a simplified control plane that runs locally and orchestrates:
1. **GitHub** (Source validation & triggering)
2. **AWS CodeBuild** (Building Docker images from Git)
3. **AWS ECR** (Storing Docker images)
4. **AWS EKS / Kubernetes** (Zero-downtime rolling deployments)

All complexity has been condensed into two main backend files, making it extremely easy to understand and present:
- `main.go`: Handles the REST API, state management, and serves the embedded UI.
- `deploy.go`: Handles AWS SDK interactions (CodeBuild) and Kubernetes orchestration (`kubectl apply`).

## Features
- **Modern Web Dashboard**: Real-time polling of deployment status via a beautiful HTML/CSS frontend.
- **Smart Fallbacks**: If AWS credentials are not configured, it gracefully falls back to local `mock` builds and local image deployment.
- **Local Kubernetes Support**: Automatically detects local `kind` or `minikube` clusters and provides smart timeouts for `LoadBalancer` IPs.

## How to Run

1. **Start the Server**
   ```powershell
   go build -o bin/orbit-server.exe .
   .\bin\orbit-server.exe
   ```

2. **Access the Dashboard**
   Open your browser to `http://localhost:8080/`.

3. **Deploy an Application**
   You can trigger deployments directly from the web dashboard or via API:
   ```powershell
   $body = @{
       repo_url = "https://github.com/your-username/orbit-test-app"
       app_name = "flask-simple-test"
   } | ConvertTo-Json

   Invoke-RestMethod -Method Post -Uri "http://localhost:8080/api/deployments" -Body $body -ContentType "application/json"
   ```

4. **Port Forwarding (Local Clusters)**
   If you are testing on a local `kind` cluster without a cloud LoadBalancer, you can access your deployed app by running:
   ```powershell
   kubectl port-forward -n orbit-apps svc/flask-simple-test 5056:8080
   ```
   Then visit `http://localhost:5056`.
