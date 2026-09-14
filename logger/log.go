package logger

import (
	"encoding/json"
	"fmt"
	"os"
)

// LogInfo is the structured log payload Gauge expects from plugins.
type LogInfo struct {
	LogLevel string `json:"logLevel"`
	Message  string `json:"message"`
}

func write(info *LogInfo) {
	b, _ := json.Marshal(info)
	fmt.Print(string(b))
}

func Debug(message string, args ...interface{}) {
	write(&LogInfo{LogLevel: "debug", Message: fmt.Sprintf(message, args...)})
}

func Info(message string, args ...interface{}) {
	write(&LogInfo{LogLevel: "info", Message: fmt.Sprintf(message, args...)})
}

func Warn(message string, args ...interface{}) {
	write(&LogInfo{LogLevel: "warning", Message: fmt.Sprintf(message, args...)})
}

func Error(message string, args ...interface{}) {
	write(&LogInfo{LogLevel: "error", Message: fmt.Sprintf(message, args...)})
}

func Fatal(message string, args ...interface{}) {
	write(&LogInfo{LogLevel: "fatal", Message: fmt.Sprintf(message, args...)})
	os.Exit(1)
}
