package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
)

var hits int
var enqueued int
var done chan bool

var queueThreshold = 256
var queueSize = 1000
var processingTime = 20

func startTicker() {
	fmt.Println("Ticker started")
	ticker := time.NewTicker(time.Second)
	done = make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case <-ticker.C:
				if hits < queueThreshold && enqueued == 0 {
					// OK
					fmt.Printf("[%d hits/s, %d in queue ] 200 OK -> no queue\n", hits, enqueued)
				} else {
					if enqueued > queueSize {
						//Deny
						fmt.Printf("[%d hits/s, %d in queue ] 503 DENIED\n", hits, enqueued)
					} else {
						//Slowdown
						fmt.Printf("[%d hits/s, %d in queue ] 200 OK -> %dms delayed\n", hits, enqueued, enqueued*processingTime)
					}
				}
				hits = 0
			}
		}
	}()
}

func stopTicker() {
	done <- true
	fmt.Println("Ticker stopped")
}

func processRequest() gin.H {
	time.Sleep(time.Duration(processingTime) * time.Millisecond)
	return gin.H{
		"message": "pong",
	}
}

func main() {
	var err error

	if queueThresholdStr := os.Getenv("QUEUE_THRESHOLD"); queueThresholdStr != "" {
		if queueThreshold, err = strconv.Atoi(queueThresholdStr); err != nil {
			fmt.Errorf("Invalid QUEUE_THRESHOLD : %s\n", err)
			os.Exit(1)
		}
	}

	if queueSizeStr := os.Getenv("QUEUE_SIZE"); queueSizeStr != "" {
		if queueSize, err = strconv.Atoi(queueSizeStr); err != nil {
			fmt.Errorf("Invalid QUEUE_SIZE : %s\n", err)
			os.Exit(1)
		}
	}

	if processingTimeStr := os.Getenv("PROCESSING_TIME"); processingTimeStr != "" {
		if processingTime, err = strconv.Atoi(processingTimeStr); err != nil {
			fmt.Errorf("Invalid PROCESSING_TIME : %s\n", err)
			os.Exit(1)
		}
	}

	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/ping", func(c *gin.Context) {
		hits++
		if hits < queueThreshold && enqueued == 0 {
			// OK
			c.JSON(200, processRequest())
		} else {
			if enqueued >= queueSize {
				//Deny
				c.AbortWithStatus(http.StatusServiceUnavailable)
			} else {
				//Slowdown
				enqueued++
				time.Sleep(time.Duration(enqueued*processingTime) * time.Millisecond)
				c.JSON(200, processRequest())
				if enqueued > 0 {
					enqueued--
				}
			}
		}
	})

	server := endless.NewServer(":8080", router)
	startTicker()
	server.SignalHooks[endless.PRE_SIGNAL][syscall.SIGHUP] = append(server.SignalHooks[endless.PRE_SIGNAL][syscall.SIGHUP], func() { stopTicker() })
	server.SignalHooks[endless.PRE_SIGNAL][syscall.SIGINT] = append(server.SignalHooks[endless.PRE_SIGNAL][syscall.SIGINT], func() { stopTicker() })
	server.SignalHooks[endless.PRE_SIGNAL][syscall.SIGTERM] = append(server.SignalHooks[endless.PRE_SIGNAL][syscall.SIGTERM], func() { stopTicker() })
	fmt.Printf("Starting server - QUEUE_THRESHOLD : %d, QUEUE_SIZE : %d, PROCESSING_TIME : %dms\n", queueThreshold, queueSize, processingTime)
	server.ListenAndServe()
}
