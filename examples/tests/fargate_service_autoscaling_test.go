package tests

import (
  "fmt"
  "testing"
  "http"
  "time"

  aws "github.com/gruntwork-io/terratest/modules/aws"
  aws_sdk "github.com/aws/aws-sdk-go/aws"
  http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
  "github.com/gruntwork-io/terratest/modules/logger"
  "github.com/gruntwork-io/terratest/modules/terraform"
  "github.com/stretchr/testify/assert"
)

// TestAutoScalingCapabilities ...
func TestAutoScalingCapabilities(t *testing.T) {
  t.Parallel()
  
  // Timeouts
  ReadinessTimeout time.Duration  = 120 * time.Second
  scaleTimeout time.Duration      = 300 * time.Second
  
  // Terraform Variables
  clusterName := "stress-demo"
  serviceName := "stress-service"
  region      := "us-east-1"

  terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
    // Set the path to the Terraform code that will be tested.
    TerraformDir: "../fargate-service-terraform",
    Vars: map[string]interface{}{
      "aws_region":   region,
      "cluster_name": clusterName,
      "service_name": serviceName,
    },
  })
  // Clean up resources with "terraform destroy" at the end of the test.
  defer terraform.Destroy(t, terraformOptions)

  // Run "terraform init" and "terraform apply". Fail the test if there are any errors.
  terraform.InitAndApply(t, terraformOptions)

  // Run `terraform output` to get the values of output variables and check they have the expected values.
  hostname := terraform.Output(t, terraformOptions, "lb_dns_name")
  // Root path used as healcheck to define if service is up & ready to receive
  url := fmt.Sprintf("http://%s", hostname)

  logger.Logf(t, "Sleep for %v as default readiness timeout.", ReadinessTimeout)
  time.Sleep(ReadinessTimeout)

  // Get cluster info
  cluster := aws.GetEcsCluster(t, region, clusterName)
  active := aws_sdk.Int64Value(cluster.ActiveServicesCount)
  assert.Equal(t, int64(1), active)
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

  service := aws.GetEcsService(t, Region, clusterName, serviceName)
  // Start the CPU load simulation and wait upscaleTimeout
  resp, _ := http.Get(fmt.Sprintf("%v/simulation/start", hostname))
  time.Sleep(scaleTimout)

  // Get service running count after the upscale
  serviceRunningCount := aws_sdk.Int64Value(service.RunningCount)
  assert.Equal(t, int64(5), serviceRunningCount)
  // Stop the CPU load simulation and wait downscaleTimeout
  resp, _ := http.Get(fmt.Sprintf("%v/simulation/stop", hostname))
  time.Sleep(scaleTimout)
  // Retreive service running count after the downscale
  serviceRunningCount := aws_sdk.Int64Value(service.RunningCount)
  assert.Equal(t, int64(5), serviceRunningCount)
}