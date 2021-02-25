package tests

import (
	"fmt"
	"testing"
	"time"

	aws_sdk "github.com/aws/aws-sdk-go/aws"
	aws "github.com/gruntwork-io/terratest/modules/aws"
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

// TestAutoScalingCapabilities ...
func TestAutoScalingCapabilities(t *testing.T) {
	t.Parallel()

	// createdTime will use instance local time to create unique service
	createdTime := time.Now().Format("150405")
	// Timeouts
	scaleTimeout := 240 * time.Second
	readinessTimeout := 120 * time.Second

	// Terraform Variables
	region := "us-east-1"
	serviceName := "stress-service-" + createdTime
	clusterName := "stress-" + createdTime + "-cluster"

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		// Set the path to the Terraform code that will be tested.
		TerraformDir: "../examples/fargate-service-terraform",
		Vars: map[string]interface{}{
			"service_name": serviceName,
			"cluster_name": clusterName,
			"aws_region":   region,
		},
	})
	// Clean up resources with "terraform destroy" at the end of the test.
	defer terraform.Destroy(t, terraformOptions)

	// Run "terraform init" and "terraform apply". Fail the test if there are any errors.
	terraform.InitAndApply(t, terraformOptions)

	// Run `terraform output` to get the values of output variables and check they have the expected values.
	hostname := terraform.Output(t, terraformOptions, "lb_dns_name")

	url := fmt.Sprintf("http://%s", hostname)

	logger.Logf(t, "Sleep for %v as default readiness timeout.", readinessTimeout)
	time.Sleep(readinessTimeout)
	// Do a simple http request to the service as healthcheck
	http_helper.HttpGetWithRetry(
		t,
		url,
		nil,
		200,
		"{\"message\":\"stress-service\",\"status\":\"ok\"}",
		5,
		15*time.Second,
	)
	// Lookup the ECS Cluster by cluster_name
	cluster := aws.GetEcsCluster(t, region, clusterName)

	activeCount := aws_sdk.Int64Value(cluster.ActiveServicesCount)
	assert.Equal(t, int64(1), activeCount)
	logger.Logf(t, "Active service count is %s", fmt.Sprint(activeCount))

	//Lookup the ECS Service
	service := aws.GetEcsService(t, region, clusterName, serviceName)

	// Start the CPU load simulation and wait upscaleTimeout
	http_helper.HttpGetWithRetry(
		t,
		url+"/simulation/start",
		nil,
		200,
		"{\"message\":\"Simulation CPU load started.\",\"status\":\"started\"}",
		5,
		15*time.Second,
	)
	logger.Logf(t, "Sleep for %v as sclae activity timeout.", fmt.Sprint(scaleTimeout))
	time.Sleep(scaleTimeout)
	// Get service running count after the upscale
	serviceRunningCount := aws_sdk.Int64Value(service.RunningCount)
	assert.Equal(t, int64(2), serviceRunningCount)

	// Stop the CPU load simulation and wait downscaleTimeout
	http_helper.HttpGetWithRetry(
		t,
		url+"/simulation/stop",
		nil,
		200,
		"{\"message\":\"Simulation CPU load has been stopped by signal\",\"status\":\"stopped\"}",
		5,
		15*time.Second,
	)
	logger.Logf(t, "Sleep for %v as sclae activity timeout.", fmt.Sprint(scaleTimeout))
	time.Sleep(scaleTimeout)
	// Retreive service running count after the downscale
	serviceRunningCount = aws_sdk.Int64Value(service.RunningCount)
	assert.Equal(t, int64(1), serviceRunningCount)
}
