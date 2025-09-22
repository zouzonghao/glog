package utils

import (
	"log"
	"os"
	"sync"
)

var (
	aiLogger *log.Logger
	logFile  *os.File
	once     sync.Once
)

func InitAILogger() {
	once.Do(func() {
		var err error
		logFile, err = os.OpenFile("ai.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatalf("无法打开 AI 日志文件: %v", err)
		}
		aiLogger = log.New(logFile, "", log.LstdFlags)
	})
}

func AILog(format string, v ...interface{}) {
	if aiLogger == nil {
		InitAILogger()
	}
	aiLogger.Printf(format, v...)
}

func CloseAILogger() {
	if logFile != nil {
		logFile.Close()
	}
}
