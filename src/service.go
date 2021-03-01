package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	status string = "stopped"
	cfg    Config = Config{
		TestTimeSeconds: 0,
		PercentageCPU:   0,
		StartAfter:      0,
	}
	stop = make(chan struct{})
)

const (
	// StatusStarted constant to define started status
	StatusStarted = "started"
	// StatusStopped constant to define stopped status
	StatusStopped = "stopped"
)

type (
	// Config ...
	Config struct {
		TestTimeSeconds int `json:"test_time_seconds"`
		PercentageCPU   int `json:"percentage_cpu"`
		StartAfter      int `json:"start_after"`
	}
	// Response ...
	Response struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	}
)

func main() {
	r := gin.Default()
	r.Use(gin.Logger())
	// Lookup environment variables for configuration
	cfg.TestTimeSeconds, _ = strconv.Atoi(os.Getenv("TEST_TIME_SECONDS"))
	if cfg.TestTimeSeconds == 0 {
		cfg.TestTimeSeconds = 600
	}
	cfg.PercentageCPU, _ = strconv.Atoi(os.Getenv("PERCENTAGE_CPU"))
	if cfg.PercentageCPU == 0 {
		cfg.PercentageCPU = 80
	}
	// Detect if user has set timer to start test automatically on startup
	cfg.StartAfter, _ = strconv.Atoi(os.Getenv("START_AFTER"))

	r.GET("/", func(ctx *gin.Context) {
		// returns the name of the service itself and status for health-check if needed
		ctx.JSON(http.StatusOK, Response{Message: "stress-service", Status: "ok"})
	})
	sim := r.Group("/simulation") // /simulation
	{
		sim.GET("/start", startHandler) // GET /simulation/start
		sim.GET("/stop", stopHandler)   // GET /simulation/stop
		// Configurations routes to read and update default config
		sim.GET("/config", func(ctx *gin.Context) { // GET /simulation/config
			ctx.JSON(http.StatusOK, cfg)
		})
		sim.POST("/config", setConfigHandler) // POST /simulation/config
	}
	// Start the service
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	if cfg.StartAfter > 0 {
		go delayedStart(cfg.StartAfter)
	}
	r.Run(fmt.Sprintf(":%v", port))
}

func startHandler(ctx *gin.Context) {
	if status != StatusStarted {
		stop = make(chan struct{})
		go runCPULoad(cfg.TestTimeSeconds, cfg.PercentageCPU)
		ctx.JSON(http.StatusOK, Response{
			Message: "Simulation CPU load started.",
			Status:  status,
		})
		return
	}
	ctx.JSON(http.StatusOK, Response{
		Message: "Simulation CPU load already in progress.",
		Status:  status,
	})
}

func stopHandler(ctx *gin.Context) {
	if status != StatusStopped {
		// Send stop signal
		close(stop)
		<-stop
		ctx.JSON(http.StatusOK, Response{
			Message: "Simulation CPU load has been stopped by signal.",
			Status:  status,
		})
		return
	}
	ctx.JSON(http.StatusOK, Response{
		Message: "Simulation CPU load was not initialized.",
		Status:  status,
	})
}

func setConfigHandler(ctx *gin.Context) {
	c := new(Config)
	if err := ctx.Bind(c); err != nil {
		ctx.JSON(http.StatusBadRequest, Response{
			Message: "Unable to bind retrived JSON data.",
			Status:  "error",
		})
		return
	}
	if c.TestTimeSeconds != 0 {
		cfg.TestTimeSeconds = c.TestTimeSeconds
	}
	if c.PercentageCPU != 0 {
		cfg.PercentageCPU = c.PercentageCPU
	}
	ctx.JSON(http.StatusCreated, cfg)
}

func runCPULoad(timeSeconds int, percentage int) {
	status = StatusStarted
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)
	unitMs := 1000 // 1 unit = 100 ms may be the best
	runMs := unitMs * percentage
	sleepMs := unitMs*100 - runMs
	log.Println("Simulating CPU load has been started.")
	// This loop will load all available cores
	for i := 1; i <= numCPU; i++ {
		go func() {
			// LockOSThread wires the calling goroutine to its current operating system thread.
			runtime.LockOSThread()
			for { // endless loop
				begin := time.Now()
				select {
				case <-stop:
					log.Println("Simulating CPU load has been stopped by signal on all cores.")
					return
				default:
					for {
						if time.Now().Sub(begin) > time.Duration(runMs)*time.Microsecond {
							break
						}
					}
				}
				time.Sleep(time.Duration(sleepMs) * time.Microsecond)
			}
		}()
	}
	time.Sleep(time.Duration(timeSeconds) * time.Second)
	status = StatusStopped
	// Send close signal to stop all running simulations
	close(stop)
	<-stop
	log.Println("Simulating CPU has been ended by timeout.")
}

func delayedStart(delaySeconds int) {
	log.Printf("Was found configuration to start CPU load automatically in %v seconds.\n", delaySeconds)
	timer := time.NewTimer(time.Duration(delaySeconds) * time.Second)
	<-timer.C
	go runCPULoad(cfg.TestTimeSeconds, cfg.PercentageCPU)
}
