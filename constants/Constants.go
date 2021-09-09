// Константы проекта
package constants

const Env = "DEV"

const (
	APP_NAME               = "OriServer for DSDOS"
	APP_VERSION            = "1.0.1"
	CONFIG_FILE            = "config.json"
	LOG_FILE               = "OriServer.log"
	ORI_DIR_MAX_FILES      = 256
	ORI_FILE_MAX_SIZE      = 65280
	ORI_DIR_MAX_SIZE       = ORI_FILE_MAX_SIZE * ORI_DIR_MAX_FILES
	ORI_DIR_MAX_SIZE_K     = ORI_DIR_MAX_SIZE >> 10
	PROTOCOL_VERSION_MAJOR = 3
	PROTOCOL_VERSION_MINOR = 0x10
)

// Типы файлов в каталоге
const (
	FT_DIR      = 0
	FT_FILE_ORI = 1
	FT_FILE_BRU = 2
	FT_FILE_ORD = 3
)

const (
	EXT_ORI = ".ORI"
	EXT_BRU = ".BRU"
	EXT_ORD = ".ORD"
)

var FILE_TYPE_NAME = map[byte]string{FT_DIR: "DIR", FT_FILE_BRU: "BRU", FT_FILE_ORD: "ORD", FT_FILE_ORI: "ORI"}

// Заголовок ORI-файла
var ORI_HEADER = []byte("Orion-128 file\r\n")

// Таблица перекодировки из UTF8 в КОИ-7
var ENCODE_TO_KOI7 = map[rune]rune{'Ю': '@', 'А': 'A', 'Б': 'B', 'Ц': 'C', 'Д': 'D',
	'Е': 'E', 'Ф': 'F', 'Г': 'G', 'Х': 'H', 'И': 'I', 'Й': 'J', 'К': 'K',
	'Л': 'L', 'М': 'M', 'Н': 'N', 'О': 'O', 'П': 'P', 'Я': 'Q', 'Р': 'R',
	'С': 'S', 'Т': 'T', 'У': 'U', 'Ж': 'V', 'В': 'W', 'Ь': 'X', 'Ы': 'Y',
	'З': 'Z', 'Ш': '[', 'Э': '\\', 'Щ': ']', 'Ч': '^', 'Ъ': '_'}

// Константы протокола
const (
	CONFIRM_ERROR          = 0x01
	CONFIRM_OK             = 0x00
	CONFIRM_DIR_DIFF       = 0x02
	CONFIRM_DIR_MISMATCH   = 0x04
	CONFIRM_TOO_MANY_FILES = 0x02
	CONFIRM_COMM_ERROR     = 0x05
	CONFIRM_FILE_EXISTS    = 0x08

	CONFIRM_PING = 0xCD

	CONFIRM_READY       = 0xED
	CONFIRM_SIZE_OK     = 0xFB
	CONFIRM_ERROR_CS    = 0xFB
	CONFIRM_DIR_OK      = 0xFD
	CONFIRM_REJECT      = 0xFE
	CONFIRM_OTHER_ERROR = 0xFF

	MARKER_DEL = 0xE5

	MSG_ILLEGAL_CODE   = "Недопустимый код подтверждения %X!"
	MSG_ILLEGAL_DIR_CS = "Ошибка контрольной суммы каталога!"

	CMD_LINKS_LOAD = 0x1D
	CMD_PING       = 0xDC
	CMD_ORI        = 0xDE
)

// Операции файлового протокола
const (
	OP_SEND_DIR        = 0x01
	OP_RECEIVE_DIR     = 0x02
	OP_SEND_FILE       = 0x03
	OP_RECEIVE_FILE    = 0x04
	OP_SEND_DIR_INFO   = 0x05
	OP_SEND_DIR_STATUS = 0x07
	OP_RECEIVE_CHANGES = 0x08
	OP_SEND_VERSION    = 0xD0
	OP_SEND_DATE       = 0xD1
	OP_SEND_TIME       = 0xD2
	OP_FILE            = 0xDF
	OP_CLIENT_REG      = 0xE0
	OP_CHK_SERVER_REG  = 0xE8

	OP_SEND_DIR_INFO_USED = 0x00
	OP_SEND_DIR_INFO_FREE = 0x01
	OP_SEND_DIR_INFO_SIZE = 0x03
)

// Тип платформы клиента
const (
	CLIENT_ORION_128     = 0
	CLIENT_SPECIALIST    = 1
	CLIENT_RADIO_86RK    = 2
	CLIENT_UT_88         = 3
	CLIENT_MICRO_80      = 4
	CLIENT_PARTNER_01_01 = 5
	CLIENT_MICROSHA      = 6
	CLIENT_ZX_SPECTRUM   = 8
	CLIENT_LIONTARY      = 0x0D
	CLIENT_ORION_PRO     = 0x0F
)

// Название платформы клиента
var CLIENT_PLATFORM = map[byte]string{
	CLIENT_ORION_128:     "Орион-128",
	CLIENT_SPECIALIST:    "Специалист",
	CLIENT_RADIO_86RK:    "Радио-86РК",
	CLIENT_UT_88:         "ЮТ-88",
	CLIENT_MICRO_80:      "Микро-80",
	CLIENT_PARTNER_01_01: "Партнёр-01.01",
	CLIENT_MICROSHA:      "Микроша",
	CLIENT_ZX_SPECTRUM:   "ZX-Spectrum",
	CLIENT_LIONTARY:      "Liontary",
	CLIENT_ORION_PRO:     "Орион-ПРО"}

//var DATA_BUTS = {5,6,7,8}
//var STOP_BITS = map[string]int{"1":1,"1.5":15,"2":2}
