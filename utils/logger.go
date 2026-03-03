package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var logger *log.Logger
var currentDate string

func init() {
	setupLogger()
}

func setupLogger() {
	if err := os.MkdirAll("logs", os.ModePerm); err != nil {
		fmt.Printf("Failed to create log directory: %v\n", err)
		return
	}

	currentDate = time.Now().Format("02-01-2006")
	logPath := filepath.Join("logs", fmt.Sprintf("mj-[%s].log", currentDate))

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		return
	}

	logger = log.New(file, "", log.LstdFlags)
}

func checkDateRotation() {
	newDate := time.Now().Format("02-01-2006")
	if newDate != currentDate {
		setupLogger()
	}
}

// Info logs generic info messages
func Info(format string, v ...interface{}) {
	checkDateRotation()
	msg := fmt.Sprintf(format, v...)
	if logger != nil {
		logger.Printf("[INFO] %s\n", msg)
	}
	fmt.Printf("[INFO] %s\n", msg)
}

// Error logs error messages
func Error(format string, v ...interface{}) {
	checkDateRotation()
	msg := fmt.Sprintf(format, v...)
	if logger != nil {
		logger.Printf("[ERROR] %s\n", msg)
	}
	fmt.Printf("[ERROR] %s\n", msg)
}
