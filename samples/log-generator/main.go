package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"
)

type logPayload struct {
	TS        string `json:"ts"`
	Level     string `json:"level"`
	Service   string `json:"service"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

var (
	levels   = []string{"info", "warn", "error"}
	messages = []string{
		"heartbeat ok",
		"request handled",
		"cache miss",
		"db query finished",
		"background job complete",
	}
)

func envOrDefault(key, def string) string {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	return value
}

func envFloatOrDefault(key string, def float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return def
	}
	return parsed
}

func emitLog(service string) {
	payload := logPayload{
		TS:        time.Now().UTC().Format(time.RFC3339Nano),
		Level:     levels[rand.Intn(len(levels))],
		Service:   service,
		Message:   messages[rand.Intn(len(messages))],
		RequestID: fmt.Sprintf("req_%d", 1000+rand.Intn(9000)),
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf(`{"ts":"%s","level":"error","service":"%s","message":"json marshal failed"}`+"\n",
			time.Now().UTC().Format(time.RFC3339Nano), service)
		return
	}
	fmt.Println(string(encoded))
}

func main() {
	rand.Seed(time.Now().UnixNano())

	service := envOrDefault("SERVICE_NAME", "log-generator")
	interval := envFloatOrDefault("LOG_INTERVAL", 2)
	ticker := time.NewTicker(time.Duration(interval * float64(time.Second)))
	defer ticker.Stop()

	for range ticker.C {
		emitLog(service)
	}
}
