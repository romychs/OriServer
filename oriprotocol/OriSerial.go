package oriprotocol

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/tarm/serial"
	conf "ru.romych/OriServer/config"
	c "ru.romych/OriServer/constants"
	m "ru.romych/OriServer/statemachine"
	"time"
)

var time500ms, _ = time.ParseDuration("500ms") // wait for connect

type SerialCommand byte

//
// Сообщение приемника/передатчика
//
type SerialMessage struct {
	command SerialCommand // Команда
	value   byte          // Принятый или передаваемый байт
	buff    *[]byte       // Передаваемый буфер
}

func BuildSerialMessage(command SerialCommand, value byte, buff *[]byte) SerialMessage {
	return SerialMessage{command, value, buff}
}

func (sm *SerialMessage) Command() SerialCommand {
	return sm.command
}

func (sm *SerialMessage) Value() byte {
	return sm.value
}

func (sm *SerialMessage) Buff() *[]byte {
	return sm.buff
}

func (sm *SerialMessage) CmdName() string {
	cn, ok := commandName[sm.Command()]
	if ok {
		return cn
	} else {
		return ""
	}
}

var (
	//conf     *config.Config // Конфигурация
	srInputChannel  chan SerialMessage // Канал для отправки команд приемнику
	srOutputChannel chan SerialMessage // Канал от приемника с принятыми данными
	swInputChannel  chan SerialMessage // Канал для отправки команд и данных передатчику
	swOutputChannel chan SerialMessage // Канал от передатчика со статусом отправки
)

//
// Инициализация асинхронных обработчиков протокола последовательного обмена
//
func InitSerialHandlers() {
	srInputChannel = make(chan SerialMessage)
	srOutputChannel = make(chan SerialMessage, 10)
	swInputChannel = make(chan SerialMessage)
	swOutputChannel = make(chan SerialMessage)

	go SerialReader(srInputChannel, srOutputChannel)
	go SerialWriter(swInputChannel, swOutputChannel)
	go SerialHandler(srOutputChannel, swOutputChannel, srInputChannel, swInputChannel)
}

//
// Возвращает количество передаваемых бит
//
//func getBits(cFlags uint32) byte {
//	if cFlags&unix.CS5 > 0 {
//		return 5
//	} else if cFlags&unix.CS6 > 0 {
//		return 6
//	} else if cFlags&unix.CS7 > 0 {
//		return 7
//	} else if cFlags&unix.CS8 > 0 {
//		return 8
//	} else {
//		return 0
//	}
//}
//
//func getStopBits(cFlags uint32) byte {
//	if cFlags&unix.CSTOPB > 0 {
//		return 2
//	} else {
//		return 1
//	}
//}
//
//func getParity(cFlags uint32) string {
//	if cFlags&unix.PARENB > 0 {
//		if cFlags&unix.PARODD > 0 {
//			return "O"
//		} else {
//			return "E"
//		}
//	} else {
//		return "N"
//	}
//}

var serialPort *serial.Port = nil

//
// Инициализирует, открывает последовательное соединение
//
func InitSerial() error {
	var err error
	serialPort, err = serial.OpenPort(getSerialConfig())
	if err != nil {
		log.Warnf("Не удалось открыть порт: %s", err.Error())
		return err
	}
	return nil
}

//
// Возвращает конфигурацию для инициализации последовательного соединения
//
func getSerialConfig() *serial.Config {
	return &serial.Config{Name: conf.Conf.SerialConfig.ComPort,
		Baud:     conf.Conf.SerialConfig.ComSpeed,
		Size:     byte(conf.Conf.SerialConfig.ComDataBits),
		Parity:   getParity(),
		StopBits: serial.StopBits(conf.Conf.SerialConfig.ComStopBits)}
}

//
// Преобразует значение четности из конфига в значение serialParity
//
func getParity() serial.Parity {
	if conf.Conf.SerialConfig.ComOdd == 1 {
		return serial.ParityOdd
	} else if conf.Conf.SerialConfig.ComOdd == 2 {
		return serial.ParityEven
	}
	return serial.ParityNone
}

//
// Закрывает последовательное соединение
//
func CloseSerial() {
	if serialPort != nil {
		err := serialPort.Close()
		if err != nil {
			log.Warnf("Не удалось нормально закрыть порт: %s", err.Error())
		}
	}
}

//
// Асинхронный приемник данных
//
func SerialReader(cmd chan SerialMessage, out chan SerialMessage) {
	buff := make([]byte, 1)
	for {
		log.Debug("SR Wait <- cmd")
		msg, ok := <-cmd
		if !ok {
			return
		}
		log.Debugf("SR Reseived cmd: %s", msg.CmdName())
		switch msg.Command() {
		case SC_STOP:
			return
		case SC_INIT:
			for {
				start := timeInMillis()
				_, err := serialPort.Read(buff)
				et := timeInMillis() - start
				if err != nil {
					if et < 500 {
						log.Warnf("Ошибка приема данных: %s", err.Error())
						out <- SerialMessage{command: SC_RECEIVE_ERROR, value: 0}
					} else {
						out <- SerialMessage{command: SC_RECEIVE_TIMEOUT, value: 0}
					}
					//break
				} else {
					//log.Infof("Received byte: %02X", buff[0])
					out <- SerialMessage{command: SC_RECEIVE_BYTE, value: buff[0]}
				}
			}

		}
	}
}

func timeInMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

//
// Асинхронный передатчик данных
//
func SerialWriter(cmd chan SerialMessage, out chan SerialMessage) {
	buff := make([]byte, 1)
	for {
		msg, ok := <-cmd
		if !ok {
			break
		}
		log.Debugf("SW Received cmd: %s; v: %02X;", msg.CmdName(), msg.value)
		switch msg.Command() {
		case SC_STOP:
			return
		case SC_INIT:
		case SC_SEND_BYTE:
			buff[0] = msg.Value()
			sendBuff(&buff, out)
		case SC_SEND_BUFF:
			sendBuff(msg.buff, out)
		}
	}
}

//
// Отправка буффера данных через последовательный порт
//
func sendBuff(buff *[]byte, out chan SerialMessage) {
	n, err := serialPort.Write(*buff)
	if err != nil {
		log.Warnf("Ошибка передачи данных: %s", err.Error())
		v := byte(0)
		if err.Error() == "EOF" {
			v = 1
		}
		out <- SerialMessage{command: SC_SEND_ERROR, value: v}
	} else if n < len(*buff) {
		out <- SerialMessage{command: SC_SEND_TIMEOUT, value: 0}
	} else {
		err = serialPort.Flush()
		if err != nil {
			log.Warnf("Ошибка flush при передаче данных: %s", err.Error())
			out <- SerialMessage{command: SC_SEND_ERROR, value: 0}
		}
		out <- SerialMessage{command: SC_SEND_OK, value: 0}
	}
}

//
// Основной обработчик протокола приема/передачи данных
//
func SerialHandler(fromSr, fromSw, toSr, toSw chan SerialMessage) {
	sm := m.Build()
	var msg SerialMessage

	defer CloseSerial()

	for {

		msg = <-fromSr

		log.Infof("Reseived cmd: %s; v: %02X;", msg.CmdName(), msg.Value())
		switch msg.Command() {
		case SC_INIT:
			err := InitSerial()
			if err == nil {
				sm.SetState(m.S_READY)
				toSr <- BuildSerialMessage(SC_INIT, 0, nil)
				toSw <- BuildSerialMessage(SC_INIT, 0, nil)
			} else {
				sm.SetState(m.S_FAIL)
				ShowConnectError(err)
			}
			SetWorkInfo("Инициализация")
		case SC_RECEIVE_BYTE:
			if sm.State() == m.S_READY {
				switch msg.Value() {
				case c.CMD_LINKS_LOAD: // Links
					sm.SetState(m.S_LINKS_LOAD)

					//loadFileAsLinks()
				case c.CMD_PING: // Ping
					sm.SetState(m.S_PING)
					toSw <- BuildSerialMessage(SC_SEND_BYTE, c.CONFIRM_PING, nil)
					anywaySetOk(fromSw, &sm)
					SetWorkInfo("Ping")
				case c.CMD_ORI:
					sm.SetState(m.S_ORI_CMD)
					toSw <- BuildSerialMessage(SC_SEND_BYTE, c.CONFIRM_READY, nil)
					checkSendOk(fromSw, &sm)
				default:
					log.Warnf("Недопустимый код операции: %02X", msg.Value())
					sm.SetState(m.S_READY)
				}
			} else if sm.State() == m.S_ORI_CMD {
				switch msg.Value() {
				case c.OP_SEND_VERSION:
					SetWorkInfo("Отправка версии")
					SendVersion(fromSw, toSw, &sm)
				case c.OP_SEND_DATE:
					SetWorkInfo("Отправка даты")
					sm.SetState(m.S_ORI_CMD_SEND_DATE)
				case c.OP_SEND_TIME:
					SetWorkInfo("Отправка даты")
					sm.SetState(m.S_ORI_CMD_SEND_TIME)
				case c.OP_FILE:
					sm.SetState(m.S_ORI_CMD_FILE)
					//opFile()
				case c.OP_CLIENT_REG:
					sm.SetState(m.S_ORI_CMD_CLIENT_REG)
				case c.OP_CHK_SERVER_REG:
					sm.SetState(m.S_ORI_CMD_CHK_SERVER_REG)
				default:
					SetWorkInfo("Недопустимая операция")
					log.Warnf("Недопустимый субкод операции CMD_ORI: %02X", msg.Value())
					sm.SetState(m.S_READY)
				}
			} else if sm.State() == m.S_ORI_CMD_SEND_DATE {
				if checkVersion(fromSw, toSw, &sm, msg.Value()) {
					toSw <- BuildSerialMessage(SC_SEND_BUFF, 0, BuildDateResponse())
					anywaySetOk(fromSw, &sm)
				}
			} else if sm.State() == m.S_ORI_CMD_SEND_TIME {
				if checkVersion(fromSw, toSw, &sm, msg.Value()) {
					toSw <- BuildSerialMessage(SC_SEND_BUFF, 0, BuildTimeResponse())
					anywaySetOk(fromSw, &sm)
				}
			} else if sm.State() == m.S_ORI_CMD_CLIENT_REG {
				SetWorkInfo("Регистрация клиента")
				if checkVersion(fromSw, toSw, &sm, msg.Value()) {
					toSw <- BuildSerialMessage(SC_SEND_BYTE, c.CONFIRM_READY, nil)
					if isSendOk(fromSw, &sm) {
						log.Debug("Wait for clientId")
						next := <-fromSr
						if next.Command() == SC_RECEIVE_BYTE {
							toSw <- BuildSerialMessage(SC_SEND_BYTE, BuildRegResponse(next.Value()), nil)
							anywaySetOk(fromSw, &sm)
						} else {
							sm.SetState(m.S_READY)
						}
					}
				}
			} else if sm.State() == m.S_ORI_CMD_CHK_SERVER_REG {
				SetWorkInfo("Проверка регистрации сервера")
				if checkVersion(fromSw, toSw, &sm, msg.Value()) {
					toSw <- BuildSerialMessage(SC_SEND_BYTE, c.CONFIRM_OK, nil)
					anywaySetOk(fromSw, &sm)
				}
			} else if sm.State() == m.S_ORI_CMD_FILE {
				if checkVersion(fromSw, toSw, &sm, msg.Value()) {
					if sendByte(fromSw, toSw, &sm, c.CONFIRM_READY) {
						OpFile(fromSr, fromSw, toSw, &sm)
					}
				}
			} else {
				log.Warnf("Недопустимое состояние CMD_ORI: %02X", msg.Value())
				sm.SetState(m.S_READY)
			}
		case SC_DISCONNECT:
			SetWorkInfo("Рассоединение")
			// Закрываем соединение
			sm.SetState(m.S_START)
			CloseSerial()
		case SC_RECEIVE_ERROR:
			SetWorkInfo("Ошибка приема данных")
			// Прервалось чтение данных, все заквываем и выходим
			sm.SetState(m.S_FAIL)
			ShowConnectError(errors.New("INPUT_ERROR"))
		case SC_RECEIVE_TIMEOUT:
			// TODO
			SetWorkInfo("Таймаут")
			log.Debug("Таймаут получения данных")
		case SC_STOP:
			SetWorkInfo("Останов")
			toSr <- BuildSerialMessage(SC_STOP, 0, nil)
			toSw <- BuildSerialMessage(SC_STOP, 0, nil)
			CloseSerial()
			return
		}
	}

}

//
// Если произошел сбой при отправке данных, возвращаем машину состояний в
// в состояние READY, для обработки новой команды
//
func checkSendOk(fromSw chan SerialMessage, sm *m.StateMachine) {
	r := <-fromSw
	if r.command != SC_SEND_OK {
		sm.SetState(m.S_READY)
	}
}

//
// Если отправлено успешно, возвращает true,
// иначе, false и сбрасывает машину состояний в состояние READY
//
func isSendOk(fromSw chan SerialMessage, sm *m.StateMachine) bool {
	r := <-fromSw
	if r.command != SC_SEND_OK {
		sm.SetState(m.S_READY)
		return false
	}
	return true
}

//
// Получаем результат отправки и какой бы он ни был, возвращаем машину
// состояний в READY, для приема новой команды
//
func anywaySetOk(fromSw chan SerialMessage, sm *m.StateMachine) {
	_ = <-fromSw
	sm.SetState(m.S_READY)
}

//
// Проверяет полученый номер версии протокола, пепедает код ошибки Ориону,
// если версия не поддерживается
//
func checkVersion(fromSw, toSw chan SerialMessage, sm *m.StateMachine, version byte) bool {
	if version > c.PROTOCOL_VERSION_MAJOR {
		toSw <- BuildSerialMessage(SC_SEND_BYTE, c.CONFIRM_OTHER_ERROR, nil)
		anywaySetOk(fromSw, sm)
		return false
	}
	return true
}

//
// Отправляет команду об ошибке в последовательный порт
//
func sendError(fromSw, toSw chan SerialMessage, sm *m.StateMachine) {
	toSw <- BuildSerialMessage(SC_SEND_BYTE, c.CONFIRM_ERROR, nil)
	anywaySetOk(fromSw, sm)
}

//
// Отправляет команду отправки байта в последовательный порт
//
func sendByte(fromSw, toSw chan SerialMessage, sm *m.StateMachine, value byte) bool {
	toSw <- BuildSerialMessage(SC_SEND_BYTE, value, nil)
	return isSendOk(fromSw, sm)
}

//
// Отправляет команду отправки массива байт в последовательный порт
//
func sendBytes(fromSw, toSw chan SerialMessage, sm *m.StateMachine, value *[]byte) bool {
	toSw <- BuildSerialMessage(SC_SEND_BUFF, 0, value)
	return isSendOk(fromSw, sm)
}

//
// Принимает байт из последовательного порта и возвращет его и true, если все ok
//
func receiveByte(fromSr chan SerialMessage, sm *m.StateMachine) (byte, bool) {
	next := <-fromSr
	if next.command == SC_RECEIVE_BYTE {
		return next.value, true
	}
	sm.SetState(m.S_READY)
	return 0, false
}

//
// Принимает указанный объем байт и расчитывает контрольную сумму
// взвращает, буффер байт, контрольную сумму и true, если все ok
//
func receiveBytes(fromSr chan SerialMessage, sm *m.StateMachine, count uint16) (*[]byte, byte, bool) {
	var buff []byte
	var checkSum byte = 0
	for ctr := uint16(0); ctr < count; ctr++ {
		next := <-fromSr
		if next.command != SC_RECEIVE_BYTE {
			sm.SetState(m.S_READY)
			return &buff, checkSum, false
		}
		checkSum ^= next.value
		buff = append(buff, next.value)
	}
	return &buff, checkSum, true
}

//
// Аосылает сигнал обработчикам на звкрытие порта
//
func DisconnectSerial() {
	srOutputChannel <- BuildSerialMessage(SC_DISCONNECT, 0, nil)
	time.Sleep(time500ms)
}

//
// Посылает сигнал обработчикам на обмен данными
//
func StartSerialHandlers() {
	srOutputChannel <- BuildSerialMessage(SC_INIT, 0, nil)
	time.Sleep(time500ms)
}

//
// Останавливает все обработчики перед завершением приложения
//
func StopSerialHandlers() {
	srOutputChannel <- BuildSerialMessage(SC_STOP, 0, nil)
	time.Sleep(time500ms)
}
