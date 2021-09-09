package oriprotocol

import (
	log "github.com/sirupsen/logrus"
	c "ru.romych/OriServer/constants"
	m "ru.romych/OriServer/statemachine"
	"time"
)

//
// Отправляет версию протокола Ориону
//
func SendVersion(fromSw, toSw chan SerialMessage, sm *m.StateMachine) {
	sm.SetState(m.S_ORI_CMD_SEND_VER)
	toSw <- BuildSerialMessage(SC_SEND_BUFF, 0, &[]byte{c.PROTOCOL_VERSION_MAJOR, c.PROTOCOL_VERSION_MINOR})
	anywaySetOk(fromSw, sm)
}

//
// Подготавливает ответ с текущей датой
//
func BuildDateResponse() *[]byte {
	year, mon, day := time.Now().Date()

	if year < 2000 {
		year = 99
	} else {
		year -= 2000
	}

	var checkSum byte = 0
	checkSum = checkSum ^ byte(day) ^ byte(mon) ^ byte(year)

	log.Infof("Отправлена текущая дата %02d.%02d.%02d", day, mon, year)

	var resp = []byte{c.CONFIRM_READY, byte(day), byte(mon), byte(year), checkSum}
	return &resp
}

//
// Подготавливает ответ с текущим временем
//
func BuildTimeResponse() *[]byte {
	hours, minutes, seconds := time.Now().Clock()

	var checkSum byte = 0
	checkSum = checkSum ^ byte(hours) ^ byte(minutes) ^ byte(seconds)
	log.Infof("Отправлено текущее время %2d:%2d:%2d", hours, minutes, seconds)

	var resp = []byte{c.CONFIRM_READY, byte(hours), byte(minutes), byte(seconds), checkSum}
	return &resp
}

func BuildRegResponse(clientId byte) byte {
	clientPlatform, found := c.CLIENT_PLATFORM[clientId&0x0F]
	if found {
		var clientCPU string
		if (clientId & 0x80) != 0 {
			clientCPU = "Z80"
		} else {
			clientCPU = "i8080"
		}
		clientRAM := uint16(((clientId&0x70)>>4)+1) * 64

		switch clientId & 0x0F {
		case c.CLIENT_ORION_128, c.CLIENT_SPECIALIST, c.CLIENT_RADIO_86RK, c.CLIENT_UT_88, c.CLIENT_PARTNER_01_01, c.CLIENT_ORION_PRO:
			log.Infof("Зарегистрирован: %s; CPU: %s; RAM: %dK", clientPlatform, clientCPU, clientRAM)
			return c.CONFIRM_OK
		default:
			log.Warnf("Не зарегистрирован: %s; CPU: %s; RAM: %dK", clientPlatform, clientCPU, clientRAM)
			return c.CONFIRM_REJECT
		}
	} else {
		log.Warnf("Неизвестная платформа: %X;", clientId)
		return c.CONFIRM_OTHER_ERROR
	}
}
