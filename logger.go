package main

import (
	"fmt"
	"log"
	"os"
	"sync"
)

var (
	logFile  *os.File
	logMutex sync.Mutex
	logger   *log.Logger
)

func initLogger() error {
	var err error
	logFile, err = os.OpenFile("log.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	logger = log.New(logFile, "", log.LstdFlags)
	return nil
}

func logInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logMutex.Lock()
	defer logMutex.Unlock()
	fmt.Printf("[INFO] %s\n", msg)
	logger.Printf("[INFO] %s", msg)
}

func logError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logMutex.Lock()
	defer logMutex.Unlock()
	fmt.Fprintf(os.Stderr, "[ERROR] %s\n", msg)
	logger.Printf("[ERROR] %s", msg)
}

func closeLogger() {
	if logFile != nil {
		logFile.Close()
	}
}
