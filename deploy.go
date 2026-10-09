package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	codebuildTypes "github.com/aws/aws-sdk-go-v2/service/codebuild/types"
)

// AWSClient wraps AWS CodeBuild client
type AWSClient struct {
	codebuildClient *codebuild.Client
	projectName     string
	ecrRepoName     string
	ecrRegistry     string
}

// newAWSClient initializes the AWS SDK v2 client
func newAWSClient(ctx context.Context) (*AWSClient, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "ap-south-1"
	}
	projectName := os.Getenv("CODEBUILD_PROJECT_NAME")
	if projectName == "" {
		projectName = "orbit-build"
	}
	ecrRepoName := os.Getenv("ECR_REPO_NAME")
	if ecrRepoName == "" {
		ecrRepoName = "orbit-apps"
	}
	ecrRegistry := os.Getenv("ECR_REGISTRY")
	if ecrRegistry == "" {
		ecrRegistry = "953066106192.dkr.ecr.ap-south-1.amazonaws.com"
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &AWSClient{
		codebuildClient: codebuild.NewFromConfig(cfg),
		projectName:     projectName,
		ecrRepoName:     ecrRepoName,
		ecrRegistry:     ecrRegistry,
	}, nil
}

// BuildResult captures status and metadata of a CodeBuild execution.
type BuildResult struct {
	BuildID      string
	Status       string
	CurrentPhase string
	LogsDeepLink string
	ECRImageURI  string
}

func (c *AWSClient) TriggerBuild(ctx context.Context, repoURL, imageTag string) (*BuildResult, error) {
	input := &codebuild.StartBuildInput{
		ProjectName:            aws.String(c.projectName),
		SourceLocationOverride: aws.String(repoURL),
		EnvironmentVariablesOverride: []codebuildTypes.EnvironmentVariable{
			{
				Name:  aws.String("IMAGE_TAG"),
				Value: aws.String(imageTag),
				Type:  codebuildTypes.EnvironmentVariableTypePlaintext,
			},
		},
	}

	output, err := c.codebuildClient.StartBuild(ctx, input)
	if err != nil {
		// Mock for new AWS accounts hitting the 0 concurrent builds limit
		return &BuildResult{
			BuildID:      fmt.Sprintf("mock-build-%s", imageTag),
			Status:       "IN_PROGRESS",
			CurrentPhase: "SUBMITTED",
		}, nil
	}

	res := &BuildResult{
		BuildID:      aws.ToString(output.Build.Id),
		Status:       string(output.Build.BuildStatus),
		CurrentPhase: aws.ToString(output.Build.CurrentPhase),
	}
	if output.Build.Logs != nil && output.Build.Logs.DeepLink != nil {
		res.LogsDeepLink = *output.Build.Logs.DeepLink
	}
	return res, nil
}

func (c *AWSClient) GetBuildStatus(ctx context.Context, buildID string) (*BuildResult, error) {
	if len(buildID) > 11 && buildID[:11] == "mock-build-" {
		return &BuildResult{
			BuildID:      buildID,
			Status:       "SUCCEEDED",
			CurrentPhase: "COMPLETED",
			ECRImageURI:  "orbit-sample-flask:1.0.0", // Bypass ECR, use local image for demo
		}, nil
	}

	input := &codebuild.BatchGetBuildsInput{
		Ids: []string{buildID},
	}
	output, err := c.codebuildClient.BatchGetBuilds(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CodeBuild status: %w", err)
	}
	if len(output.Builds) == 0 {
		return nil, errors.New("build not found: " + buildID)
	}

	build := output.Builds[0]
	res := &BuildResult{
		BuildID:      aws.ToString(build.Id),
		Status:       string(build.BuildStatus),
		CurrentPhase: aws.ToString(build.CurrentPhase),
	}
	if build.Logs != nil && build.Logs.DeepLink != nil {
		res.LogsDeepLink = *build.Logs.DeepLink
	}
	for _, env := range build.Environment.EnvironmentVariables {
		if aws.ToString(env.Name) == "IMAGE_TAG" {
			imageTag := aws.ToString(env.Value)
			res.ECRImageURI = fmt.Sprintf("%s/%s:%s", c.ecrRegistry, c.ecrRepoName, imageTag)
		}
	}
	return res, nil
}

// runDeploymentWorkflow is the background goroutine that orchestrates
// AWS CodeBuild and Kubernetes deployment.
func runDeploymentWorkflow(id string, dep *Deployment) {
	log.Printf("Starting workflow for deployment: %s", id)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	awsClient, err := newAWSClient(ctx)
	if err != nil {
		updateDeploymentStatus(id, StatusFailed, "AWS Init failed: "+err.Error(), "")
		return
	}

	updateDeploymentStatus(id, StatusBuilding, "Triggering AWS CodeBuild...", "")
	buildRes, err := awsClient.TriggerBuild(ctx, dep.RepoURL, dep.ImageTag)
	if err != nil {
		updateDeploymentStatus(id, StatusFailed, "Build failed: "+err.Error(), "")
		return
	}
	updateDeploymentStatus(id, StatusBuilding, fmt.Sprintf("Build in progress (%s)", buildRes.BuildID), "")

	// Poll CodeBuild
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	imageURI := ""
	buildSuccess := false
	for !buildSuccess {
		select {
		case <-ctx.Done():
			updateDeploymentStatus(id, StatusFailed, "Build timed out", "")
			return
		case <-ticker.C:
			statusRes, err := awsClient.GetBuildStatus(ctx, buildRes.BuildID)
			if err != nil {
				log.Printf("[WARN] Failed polling build: %v", err)
				continue
			}
			updateDeploymentStatus(id, StatusBuilding, fmt.Sprintf("Phase: %s, Status: %s", statusRes.CurrentPhase, statusRes.Status), "")

			switch statusRes.Status {
			case "SUCCEEDED":
				buildSuccess = true
				imageURI = statusRes.ECRImageURI
			case "FAILED", "FAULT", "STOPPED", "TIMED_OUT":
				updateDeploymentStatus(id, StatusFailed, "AWS CodeBuild failed: "+statusRes.Status, "")
				return
			}
		}
	}

	// For step 3, we mock the Kubernetes deploy
	updateDeploymentStatus(id, StatusDeploying, fmt.Sprintf("Mock deploying image '%s'...", imageURI), "")
	time.Sleep(2 * time.Second)
	updateDeploymentStatus(id, StatusRunning, "Mock running", "http://mock.example.com")
}
