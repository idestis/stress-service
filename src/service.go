package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	testStatus string = "stopped"
	cfg        Config = Config{
		TestTimeSeconds: 300,
		PercentageCPU:   80,
	}
)

type (
	// Config ...
	Config struct {
		TestTimeSeconds int `json:"test_time_seconds"`
		PercentageCPU   int `json:"percentage_cpu"`
	}
	// Response ...
	Response struct {
		Data   string `json:"data"`
		Status string `json:"status"`
	}
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/", rootHandler)
	testing := e.Group("/testing")
	testing.GET("/start", startTestHandler)
	testing.GET("/stop", stopTestHandler)
	testing.POST("/config", setConfigHandler)
	testing.GET("/config", getConfigHandler)
	// Start our Service
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%v", port)))
}

func rootHandler(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, Response{Data: "stress-service", Status: "ok"})
}

func startTestHandler(ctx echo.Context) error {
	if testStatus == "started" {
		return ctx.JSON(http.StatusOK, Response{Data: "test is already active", Status: testStatus})
	}
	testStatus = "started"
	t := time.Now()
	go runCPULoad(cfg.TestTimeSeconds, cfg.PercentageCPU)
	message := fmt.Sprintf("Test started at %v with %v", t, cfg)
	return ctx.JSON(http.StatusOK, Response{Data: message, Status: testStatus})
}

func stopTestHandler(ctx echo.Context) error {
	if testStatus == "stopped" {
		return ctx.JSON(http.StatusOK, Response{Data: "test was not started", Status: testStatus})
	}
	testStatus = "stoped"
	return ctx.JSON(http.StatusOK, Response{Data: "time-and-date", Status: testStatus})
}

func setConfigHandler(ctx echo.Context) (err error) {
	c := new(Config)
	if err := ctx.Bind(c); err != nil {
		return ctx.JSON(http.StatusBadRequest, Response{
			Data:   "",
			Status: "error",
		})
	}
	if c.TestTimeSeconds != 0 {
		cfg.TestTimeSeconds = c.TestTimeSeconds
	}
	if c.PercentageCPU != 0 {
		cfg.PercentageCPU = c.PercentageCPU
	}
	return ctx.JSON(http.StatusCreated, cfg)
}

func getConfigHandler(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, cfg)
}

func runCPULoad(timeSeconds int, percentage int) {
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)
	// 1 unit = 100 ms may be the best
	unitHundresOfMicrosecond := 1000
	runMicrosecond := unitHundresOfMicrosecond * percentage
	sleepMicrosecond := unitHundresOfMicrosecond*100 - runMicrosecond
	for i := 0; i < numCPU; i++ {
		go func() {
			runtime.LockOSThread()
			for { // endless loop
				begin := time.Now()
				for {
					if time.Now().Sub(begin) > time.Duration(runMicrosecond)*time.Microsecond {
						break
					}
				}
				time.Sleep(time.Duration(sleepMicrosecond) * time.Microsecond)
			}
		}()
	}
	time.Sleep(time.Duration(timeSeconds) * time.Second)
}
