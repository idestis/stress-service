# Stress Service

[![Publish Docker Image](https://github.com/idestis/stress-service/actions/workflows/docker-image.yml/badge.svg)](https://github.com/idestis/stress-service/actions/workflows/docker-image.yml)

Current service can simulate CPU load on any environment, useful to test auto scaling capabilities on AWS or GCP. Controllable start/stop allows you to test upscale and downscale.

## Build

In case if your policy doesn't allow you to pull from GitHub registry, you can build your own image

```bash
$ docker build -t stress-service .
[+] Building 23.8s (11/15)
 => exporting to image
 => => writing image sha256:98a74165a79774c5ecc4b92a092aaf8ef3aa2f435e232236ed2a76e89846386f
 => => naming to docker.io/library/stress-service
```

Then run `docker images` to find your local image. Tag fresh image whenewer you need to publish on your private registries [using official documentation](https://docs.docker.com/engine/reference/commandline/tag/)

## Use Case

Using [terratest](https://terratest.gruntwork.io) you can validate you autoscaling capabilities of terraform module for ECS deployment

```go
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
    TerraformDir: "../examples/fargate-service-terraform",
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
```

## Service Documentation

This service contains root endpoint `/` which can be used as healthcheck and simulation specific endpoints described below.

### Simulation Configuration

* **URL**

  `/simulation/config`

* **Method**

  `GET` | `POST`

* **Data Params**

  If you are making post response to update simulation cofiguration, services required `Content-Type: application/json` and body payload with one or all parameters

  ```json
  {
    "percentage_cpu": 70,
    "test_time_seconds": 120
  }
  ```

  Where the `percentage_cpu` is how many CPU resources will be used and `test_time_seconds` is duration of test in seconds.

* **Success Response**

  * **Code** 200 <br/>
    **Content**

    ```json
    {"percentage_cpu": 70, "test_time_seconds":300 }
    ```

* **Error Response**

  * **Code** 400 <br/>
    **Content**

    ```json
    {"message": "Unable to bind retrived JSON data.", "status":"error" }
    ```

### Simulation Start

* **URL**

  `/simulation/start`

* **Method**

  `GET`

* **Success Response**

  * **Code** 200 <br/>
    **Content**

    ```json
    {"message":"Simulation CPU load started.","status":"started"}
    ```

* **Notes**

  Simulation will be automatically stoped after reaching `test_time_seconds` which is default `300` seconds.

### Simulation Stop

* **URL**

  `/simulation/stop`

* **Method**

  `GET`

* **Success Response**

  * **Code** 200 <br/>
    **Content**

    ```json
    {"message":"Simulation CPU load has been stopped by signal.","status":"started"}
    ```

## Contribute

The following information is a set of guidelines for contributing to `stress-service`. These are mostly guidelines, not rules. Use your best and feel free to propose changes using pull request or issues.

### Code of Conduct

#### **How Can I Contribute?**

* **Reporting Bugs**

  For instance, you found a bug, please refer to [GitHub Issues](https://github.com/idestis/stress-service/issues) and open a new issue with label `bug`.
  Leave the detailed description of how we can reproduce it and resolve this issue.

* **How Do I Submit A (Good) Bug Report?**

  * **Use a clear and descriptive title** for the issue to identify the problem
  * **Describe the exact steps which reproduce the problem** in as many details as possible.
  * **Describe the behavior you observed after following the steps**
  * **Explain which behavior you expected to see instead and why**

* **Suggesting Enhancements**

  Enhancement suggestions are tracked as [GitHub Issues](https://github.com/idestis/stress-service/issues)

  In case when your inner voice desire a new feature, please open an issue with `enhancement` label.

  But please refer to short rules to open a good enhancement request

  * **Use a clear and descriptive title** for the issue
  * **Describe which behavior you expect** with this feature

### **Development**

When you are good with Go, you can fork this repo and suggest changes via Pull Requests. Please read an official documentation provided by GitHub.

[docs.github.com/creating-a-pull-request-from-a-fork](https://docs.github.com/en/github/collaborating-with-issues-and-pull-requests/creating-a-pull-request-from-a-fork)
