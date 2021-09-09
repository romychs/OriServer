package main

import (
	"github.com/romychs/gotk3/gtk"
	log "github.com/sirupsen/logrus"
	c "ru.romych/OriServer/constants"
	"ru.romych/OriServer/logger"
	"ru.romych/OriServer/oriprotocol"
)

//
// Основной процесс
//
func main() {
	logger.InitLog()

	log.Infof("%s version %s started.", c.APP_NAME, c.APP_VERSION)

	oriprotocol.InitSerialHandlers()

	gtk.Init(nil)

	oriprotocol.InitMainWindow()
	oriprotocol.SetWorkInfo("Готов")
	gtk.Main()
}
