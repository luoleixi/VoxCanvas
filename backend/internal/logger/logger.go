package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func Init(logDir, logFile string) (*os.File, error) {
	if logDir == "" {
		logDir = "logs"
	}
	if logFile == "" {
		logFile = "voxcanvas-backend.log"
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	path := filepath.Join(logDir, logFile)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	writer := io.MultiWriter(os.Stdout, file)
	log.SetOutput(writer)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.LUTC)
	gin.DefaultWriter = writer
	gin.DefaultErrorWriter = writer
	log.Printf("[LOGGER] output=%s", path)
	return file, nil
}

func MaskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "..." + value[len(value)-4:]
}
