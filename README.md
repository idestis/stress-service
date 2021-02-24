# Stress Service

Service generates CPU load on any environment, useful to test auto scaling capabilities on AWS or GCP.

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

// TestAutoScalingCapabilities ...
func TestAutoScalingCapabilities(t *testing.T) {

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
    "percentage_cpu": number,
    "test_time_seconds": number
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

