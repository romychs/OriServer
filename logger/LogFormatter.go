//
// ПАО "Почта Банк"
// ДБО
//
package logger

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	c "ru.romych/OriServer/constants"
	"strconv"
	"strings"
)

type OriLogFormatter struct {
}

func (f *OriLogFormatter) Format(entry *log.Entry) ([]byte, error) {
	caller := "-"
	if entry.HasCaller() {
		caller = filepath.Base(entry.Caller.File) + ":" + strconv.Itoa(entry.Caller.Line)
	}
	lvl := strings.ToUpper(entry.Level.String())
	if len(lvl) > 0 {
		lvl = lvl[0:1]
	} else {
		lvl = "-"
	}
	logMessage := fmt.Sprintf("%s[%s][%s]: %s\n", entry.Time.Format("2006-01-02T15:04:05"), lvl, caller, entry.Message) //time.RFC3339

	return []byte(logMessage), nil
}

func InitLog() {
	log.SetFormatter(new(OriLogFormatter))
	log.SetReportCaller(true)
	if c.Env == "DEV" {
		log.SetOutput(os.Stdout)
		log.SetLevel(log.DebugLevel)
	} else {
		// Перенаправляем логи в файл
		logFile, err := os.OpenFile(c.LOG_FILE, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Не удалось открыть файл лога: %v", err)
		}
		log.SetOutput(logFile)
	}
}
