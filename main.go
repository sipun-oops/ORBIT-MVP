package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

//go:embed web/index.html
var webFS embed.FS

// DeploymentStatus constants
const (
	StatusPending   = "PENDING"
	StatusBuilding  = "BUILDING"
	StatusDeploying = "DEPLOYING"
	StatusRunning   = "RUNNING"
	StatusFailed    = "FAILED"
)

// Deployment represents a single deployment job.
type Deployment struct {
	ID        string `json:"id"`
	RepoURL   string `json:"repo_url"`
	AppName   string `json:"app_name"`
	ImageTag  string `json:"image_tag"`
	Status    string `json:"status"`
	PublicURL string `json:"public_url,omitempty"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

// Global state
var (
	deployments   = make(map[string]*Deployment)
	deploymentsMu sync.RWMutex
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// 1. Dashboard
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := webFS.ReadFile("web/index.html")
		if err != nil {
			http.Error(w, "Dashboard not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	})

	// 2. Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "app": "orbit-mvp-simple"})
	})

	// 3. API Routes
	mux.HandleFunc("/api/deployments", handleDeployments)
	// For paths like /api/deployments/{id} or /api/deployments/{id}/logs
	mux.HandleFunc("/api/deployments/", handleDeploymentDetails)

	log.Printf("Starting ORBIT MVP (Simple) on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleDeployments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		deploymentsMu.RLock()
		defer deploymentsMu.RUnlock()
		var list []*Deployment
		for _, d := range deployments {
			list = append(list, d)
		}
		json.NewEncoder(w).Encode(list)

	case http.MethodPost:
		var req struct {
			RepoURL string `json:"repo_url"`
			AppName string `json:"app_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		req.RepoURL = strings.TrimSpace(req.RepoURL)
		req.AppName = strings.TrimSpace(req.AppName)

		if err := validateGitURL(req.RepoURL); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		if req.AppName == "" {
			http.Error(w, `{"error":"app_name required"}`, http.StatusBadRequest)
			return
		}

		id := fmt.Sprintf("%s-%d", req.AppName, time.Now().Unix())
		imageTag := fmt.Sprintf("v%d", time.Now().Unix())

		dep := &Deployment{
			ID:        id,
			RepoURL:   req.RepoURL,
			AppName:   req.AppName,
			ImageTag:  imageTag,
			Status:    StatusPending,
			Message:   "Deployment queued",
			CreatedAt: time.Now().Format(time.RFC3339),
		}

		deploymentsMu.Lock()
		deployments[id] = dep
		deploymentsMu.Unlock()

		// Call the workflow loop. We will stub it temporarily if deploy.go doesn't exist yet,
		// but since it's in the same package we can call it. We'll add a dummy in deploy.go next.
		go runDeploymentWorkflow(id, dep)

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(dep)

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func handleDeploymentDetails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	path := strings.TrimPrefix(r.URL.Path, "/api/deployments/")
	parts := strings.Split(path, "/")
	id := parts[0]

	if id == "" {
		http.Error(w, `{"error":"id required"}`, http.StatusBadRequest)
		return
	}

	deploymentsMu.RLock()
	dep, exists := deployments[id]
	deploymentsMu.RUnlock()

	if !exists {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	if len(parts) == 1 {
		// GET /api/deployments/{id}
		json.NewEncoder(w).Encode(dep)
		return
	}

	if len(parts) == 2 && parts[1] == "logs" {
		json.NewEncoder(w).Encode(map[string]string{"logs": "Logs streaming not yet implemented."})
		return
	}

	http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
}

func validateGitURL(repoURL string) error {
	if repoURL == "" {
		return fmt.Errorf("repo_url required")
	}
	u, err := url.Parse(repoURL)
	if err != nil || u.Scheme != "https" {
		return fmt.Errorf("only HTTPS Git URLs are allowed")
	}
	if u.Host != "github.com" {
		return fmt.Errorf("only github.com repositories are allowed")
	}
	return nil
}

// updateDeploymentStatus safely updates the status and message.
func updateDeploymentStatus(id, status, msg, url string) {
	deploymentsMu.Lock()
	defer deploymentsMu.Unlock()
	if d, ok := deployments[id]; ok {
		d.Status = status
		d.Message = msg
		if url != "" {
			d.PublicURL = url
		}
	}
}
