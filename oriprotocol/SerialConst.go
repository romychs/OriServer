package oriprotocol

import (
	ot "ru.romych/OriServer/oritypes"
)

//
// Коды команд передатчика и приемника
//
const (
	SC_INIT            = iota // Инициализация
	SC_RECEIVE_BYTE           // Прием байта из порта
	SC_SEND_BYTE              // Отправка байта в порт
	SC_SEND_BUFF              // Передача буфера байт через порт
	SC_SEND_OK                // Статус успешной отправки данных
	SC_DISCONNECT             // Закрытие соединения
	SC_RECEIVE_TIMEOUT = 0xFB // При приеме данных возник таймаут
	SC_SEND_TIMEOUT    = 0xFC // При передаче данных возник таймаут
	SC_RECEIVE_ERROR   = 0xFD // При приеме данных возникла ошибка
	SC_SEND_ERROR      = 0xFE // При передаче данных возникла ошибка
	SC_STOP            = 0xFF // Завершить прием и передачу
)

var BAUD_RATES = ot.KeyValueArray{ot.KeyValue{"300 бод", 300}, ot.KeyValue{"600 бод", 600},
	ot.KeyValue{"1200 бод", 1200}, ot.KeyValue{"2400 бод", 2400},
	ot.KeyValue{"4800 бод", 4800}, ot.KeyValue{"9600 бод", 9600},
	ot.KeyValue{"19200 бод", 19200}, ot.KeyValue{"38400 бод", 38400},
	ot.KeyValue{"57600 бод", 57600}, ot.KeyValue{"115200 бод", 155200}}

var DATA_BITS = ot.KeyValueArray{ot.KeyValue{"6 бит", 6}, ot.KeyValue{"7 бит", 7}, ot.KeyValue{"8 бит", 8}}
var EVEN_VALUES = ot.KeyValueArray{ot.KeyValue{"None", 0}, ot.KeyValue{"Odd", 1}, ot.KeyValue{"Even", 2}}
var STOP_BITS = ot.KeyValueArray{ot.KeyValue{"1 бит", 1}, ot.KeyValue{"1,5 бита", 15}, ot.KeyValue{"2 бита", 2}}

//
// Текстовое представление команд для логирования
//
var commandName = map[SerialCommand]string{
	SC_INIT:            "SC_INIT",
	SC_RECEIVE_BYTE:    "SC_RECEIVE_BYTE",
	SC_SEND_BYTE:       "SC_SEND_BYTE",
	SC_SEND_BUFF:       "SC_SEND_BUFF",
	SC_SEND_OK:         "SC_SEND_OK",
	SC_DISCONNECT:      "SC_DISCONNECT",
	SC_RECEIVE_TIMEOUT: "SC_RECEIVE_TIMEOUT",
	SC_SEND_TIMEOUT:    "SC_SEND_TIMEOUT",
	SC_RECEIVE_ERROR:   "SC_RECEIVE_ERROR",
	SC_SEND_ERROR:      "SC_SEND_ERROR",
	SC_STOP:            "SC_STOP"}
