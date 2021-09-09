package oriprotocol

import (
	"bytes"
	"errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	conf "ru.romych/OriServer/config"
	c "ru.romych/OriServer/constants"
	t "ru.romych/OriServer/oritypes"
	m "ru.romych/OriServer/statemachine"
	"sort"
	"strings"
)

//
// Читает каталог файлов с фильтрацией по орионовским файлам
//
func ReadDir(path string) *[]t.FileInfo {
	var dir []t.FileInfo

	files, err := ioutil.ReadDir(path)

	if err != nil {
		log.Fatal(err)
	}
	if len(path) > 1 {
		dir = append(dir, t.FileInfo{"..", 0, 0, 0, c.FT_DIR})
	}
	for _, file := range files {
		if file.IsDir() {
			appendDir(file, &dir)
		} else {
			checkAndAppendFile(file, &dir)
		}
	}

	// Сортировка списка, каталоги, потом файлы по алфавиту
	sort.SliceStable(dir, func(i, j int) bool {
		if dir[i].FType == c.FT_DIR && dir[j].FType != c.FT_DIR {
			return true
		}
		if dir[j].FType == c.FT_DIR && dir[i].FType != c.FT_DIR {
			return false
		}
		return strings.ToUpper(dir[i].Name) < strings.ToUpper(dir[j].Name)
	})
	dirChanged = true
	return &dir
}

//
// Заполняет структуру каталога для Ориона
//
func buildOriDir(dir *[]t.FileInfo) (*[]byte, byte) {
	var oriDir []byte
	var header []byte
	for n, file := range *dir {
		if n == c.ORI_DIR_MAX_FILES {
			log.Warnf("В каталоге более %d файлов, часть пропущена!", c.ORI_DIR_MAX_FILES)
			break
		}
		switch file.FType {
		case c.FT_DIR:
			header = getOriDirItem(file.Name)
		case c.FT_FILE_ORI:
			h, err := readHeader(file.Name, true)
			if err != nil {
				continue
			}
			header = *h
		case c.FT_FILE_ORD, c.FT_FILE_BRU:
			h, err := readHeader(file.Name, false)
			if err != nil {
				continue
			}
			header = *h
			header[0x0D] = 0 // для BRU/ORD-файлов обнуляем байты 0x0D, 0x0E, 0x0F заголовка
			header[0x0E] = 0
			header[0x0F] = 0
		}
		oriDir = append(oriDir, header...)
	}
	var checkSum byte = 0
	for _, b := range oriDir {
		checkSum ^= b
	}
	return &oriDir, checkSum
}

var trigger = false
var dirChanged = false
var sentDirData *[]byte = nil
var dirData *[]t.FileInfo = nil

//
// Операции с файлами
//
func OpFile(fromSr, fromSw, toSw chan SerialMessage, sm *m.StateMachine) {

	cmd, ok := receiveByte(fromSr, sm)
	if ok {
		switch cmd {
		case c.OP_SEND_DIR:
			SetWorkInfo("Отправка каталога")
			opSendDir(fromSr, fromSw, toSw, sm)
		case c.OP_RECEIVE_DIR:
			SetWorkInfo("Прием каталога")
			opReceiveDir(fromSr, fromSw, toSw, sm)
		case c.OP_SEND_FILE:
			trigger = false
			SetWorkInfo("Отправка файла")
			opSendFile(fromSr, fromSw, toSw, sm)
		case c.OP_RECEIVE_FILE:
			trigger = false
			SetWorkInfo("Прием файла")
			opReceiveFile(fromSr, fromSw, toSw, sm)
		case c.OP_SEND_DIR_INFO:
			SetWorkInfo("Отправка инфо о каталоге")
			opSendDirInfo(fromSr, fromSw, toSw, sm)
		case c.OP_SEND_DIR_STATUS:
			SetWorkInfo("Отправка статуса каталога")
			opSendDirStatus(fromSw, toSw, sm)
		case c.OP_RECEIVE_CHANGES:
			SetWorkInfo("Прием изменений")
			opReceiveChanges(fromSr, fromSw, toSw, sm)
		default:
			sendByte(fromSw, toSw, sm, c.CONFIRM_OTHER_ERROR)
		}
	}
	sm.SetState(m.S_READY)
	SetWorkInfo("Готов")
}

//
// Отправляет содержимое рабочего каталога на Орион
//
func opSendDir(fromSr, fromSw, toSw chan SerialMessage, sm *m.StateMachine) {
	dirData = ReadDir(conf.Conf.WorkPath())
	if len(*dirData) > c.ORI_DIR_MAX_FILES {
		sendError(fromSw, toSw, sm)
		log.Warnf("В каталоге более %d файлов!", c.ORI_DIR_MAX_FILES)
		return
	}

	if !sendByte(fromSw, toSw, sm, c.CONFIRM_OK) { // готов
		return
	}

	dirLen := byte(len(*dirData))
	if !sendByte(fromSw, toSw, sm, dirLen) { // отправка кол-ва файлов в каталоге
		return
	}

	if dirLen > 0 {
		confirm, ok := receiveByte(fromSr, sm)
		if ok {
			switch confirm {
			case c.CONFIRM_TOO_MANY_FILES:
				log.Warn("Ответ клиента: Слишком много файлов!")
				return
			case c.CONFIRM_DIR_OK:
				log.Info("Отправка каталога.")
				oriDir, checkSum := buildOriDir(dirData)
				*oriDir = append(*oriDir, checkSum)
				toSw <- BuildSerialMessage(SC_SEND_BUFF, 0, oriDir)
				if !isSendOk(fromSw, sm) {
					return
				}

				confirm, ok := receiveByte(fromSr, sm)
				if ok {
					if confirm != c.CONFIRM_OK {
						if confirm == c.CONFIRM_ERROR {
							log.Warn(c.MSG_ILLEGAL_DIR_CS)
						} else {
							log.Warnf(c.MSG_ILLEGAL_CODE, confirm)
						}
						return
					}
					log.Info("Каталог отправлен.")
					sentDirData = oriDir
					trigger = true
				}
				dirChanged = false
			default:
				log.Warnf(c.MSG_ILLEGAL_CODE, confirm)
			}
		}
	} else {
		// пустой каталог
		confirm, ok := receiveByte(fromSr, sm)
		if ok {
			if confirm != c.CONFIRM_OK {
				log.Warnf("Неправильный код подтверждения %02X!", confirm)
			}
		}
	}
	trigger = true
	sm.SetState(m.S_READY)
}

//
// Принимает каталог от Ориона и вносит изменения в рабочем каталоге
//
func opReceiveDir(fromSr, fromSw, toSw chan SerialMessage, sm *m.StateMachine) {

	count, ok := receiveByte(fromSw, sm) // прием размера каталога
	if !ok {
		return
	}

	if count == 0 {
		log.Warn("Попытка записи пустого каталога")
		sendByte(fromSw, toSw, sm, c.CONFIRM_OK)
		trigger = false
	} else if !trigger {
		sendByte(fromSw, toSw, sm, c.CONFIRM_OK)
	} else {
		trigger = false
		//dir := *ReadDir(workPath)
		if count > byte(len(*dirData)+1) {
			sendByte(fromSw, toSw, sm, c.CONFIRM_DIR_DIFF)
			log.Warn("Несоответствие каталога!")
		} else {
			// отправка кода запроса тела каталога
			if !sendByte(fromSw, toSw, sm, c.CONFIRM_OK) {
				return
			}
			log.Info("Получение каталога.")
			savedDirData, checkSum, ok := receiveBytes(fromSr, sm, uint16(count)*t.DIR_ENTRY_LEN)
			if !ok {
				return
			}
			oriCheckSum, ok := receiveByte(fromSr, sm)
			if !ok {
				return
			}
			if oriCheckSum == checkSum {
				res, fileNo, ptr := isEqualDirData(savedDirData, sentDirData)
				if res {
					log.Info("Изменения в каталоге не обнаружены")
					sendByte(fromSw, toSw, sm, c.CONFIRM_OK)
				} else {
					file := (*dirData)[fileNo]
					buf, err := readHeader(file.Name, false)
					if err != nil {
						sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR) // ошибка записи каталога, не удалось открыть файл
					} else {
						ext := strings.ToUpper(filepath.Ext(file.Name))
						if ext == c.EXT_ORI {
							if isOriFile(file.Name) {
								var err error
								buf, err = readHeader(file.Name, true)
								if err != nil {
									sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
									return
								}
							} else {
								sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
								return
							}
						} else {
							(*buf)[0x0F] = 0 // TODO: Check if we need it
							(*buf)[0x0E] = 0
							(*buf)[0x0D] = 0
						}

						offset := fileNo * 16
						dirEntry := (*savedDirData)[offset : offset+16]
						if bytes.Equal(*buf, dirEntry) {
							op := dirEntry[0]

							if ptr == 0 && op == c.MARKER_DEL {
								// удаление файла
								deleteOriFile(fromSw, toSw, sm, file.Name)
							} else if isOriFile(file.Name) { // работаем только с ORI-файлами!

								buf, err = readHeader(file.Name, true)

								if ptr < 8 { // переименование файла
									newName := dirEntry[0:8]
									renameOriFile(fromSw, toSw, sm, file.Name, &newName)
								} else if ptr == 8 || ptr == 9 { // изменен адрес посадки файла
									//newAddr := savedDirData[8 : 10]
									log.Infof("Изменен адрес посадки файла %s на %04X", file.Name, uint16(dirEntry[8])+uint16(dirEntry[9])*256)
								} else if ptr == 10 || ptr == 11 { // изменение длины файла
									log.Warnf("Нельзя изменить длину файла %s", file.Name)
									sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
									return
								} else if ptr == 12 {
									log.Infof("Изменены атрибуты файла %s на %02X", file.Name, dirEntry[12])
								} else if ptr == 13 {
									log.Infof("Изменена страница в ОЗУ файла %s на %X", file.Name, dirEntry[13])
								} else {
									log.Infof("Изменена дата файла %s", file.Name)
									sendByte(fromSw, toSw, sm, c.CONFIRM_OTHER_ERROR)
								}

								if !writeOriHeader(file.Name, &dirEntry) {
									sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
								} else {
									sendByte(fromSw, toSw, sm, c.CONFIRM_OK)
									SetNewWorkPath(conf.Conf.WorkPath())
									//dirData = ReadDir(conf.Conf.WorkPath())
								}
							} else {
								log.Warnf("Переименование файла %s невозможно!", file.Name)
								sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
							}

						} else {
							log.Warn("Несоответствие каталога!")
							sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
						}
					}
				}
			} else {
				log.Warn(c.MSG_ILLEGAL_DIR_CS)
				sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
			}
		}
	}
}

//
// Отправляет содержимое файла на Орион или меняет каталог, если получено имя каталога
//
func opSendFile(fromSr, fromSw, toSw chan SerialMessage, sm *m.StateMachine) {
	var fileName []byte
	var fileEntry []byte
	for cnt := 0; cnt < 8; cnt++ {

		b, ok := receiveByte(fromSr, sm)
		if !ok {
			return
		}

		if b < 0x20 || b > 0x80 {
			log.Warnf("Получен недопустимый символ в имени файла! Код символа: 0x%X!", b)
			sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
			sm.SetState(m.S_READY)
			return
		}
		fileName = append(fileName, b)
	}
	// Ищем файл в каталоге
	count := byte(len(*sentDirData) / 16)
	found := false
	offset := 0
	var fileNo byte
	for fileNo = 0; fileNo < count; fileNo++ {
		fn := (*sentDirData)[offset : offset+8]
		fileEntry = (*sentDirData)[offset : offset+16]
		found = bytes.Equal(fn, fileName)
		if found {
			break
		}
		offset += 16
	}
	if !found {
		log.Warn("Каталоги различаются!")
		sendByte(fromSw, toSw, sm, c.CONFIRM_DIR_MISMATCH)
		sm.SetState(m.S_READY)
		return
	}
	name := (*dirData)[fileNo].Name

	if len(fileName) > 0 && fileName[0] == '.' { // это каталог
		changeDir(fromSr, fromSw, toSw, sm, name, fileName)
	} else {
		sendFileBody(fromSr, fromSw, toSw, sm, name, fileName, fileEntry)
	}

}

//
// Принимает содержимое файла от Ориона
//
func opReceiveFile(fromSr, fromSw, toSw chan SerialMessage, sm *m.StateMachine) {

	header := receiveHeader(fromSr, fromSw, toSw, sm)
	if header != nil {
		_, offset := findFileInDir(header)
		if offset != -1 {
			log.Warn("Файл уже существует!")
			sendByte(fromSw, toSw, sm, c.CONFIRM_FILE_EXISTS)
			return
		}
		oriFile := extractFileName(header)
		if oriFile == "" {
			log.Warn("Пустое имя выходного файла!")
			sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
			return
		}

		fileName := oriFile + c.EXT_ORI
		if !sendByte(fromSw, toSw, sm, c.CONFIRM_SIZE_OK) {
			return
		}
		log.Infof("Получение файла: %s...", fileName)
		size := uint16((*header)[10]) + uint16((*header)[11])<<8

		fileBody, checkSum, ok := receiveBytes(fromSr, sm, size)
		if !ok {
			return
		}

		oriCheckSum, ok := receiveByte(fromSr, sm)
		if !ok {
			return
		}

		if oriCheckSum != checkSum {
			sendByte(fromSw, toSw, sm, c.CONFIRM_COMM_ERROR)
			log.Warn("Ошибка контрольной суммы!")
			return
		}

		if oriFile[0] == '.' { // Папка
			if oriFile[1] == '.' && oriFile[2] == ' ' { // папка вверх
				sendByte(fromSw, toSw, sm, c.CONFIRM_COMM_ERROR)
				log.Warn("Некорректное имя каталога '..'!")
				return
			}

			if !createDirectory(oriFile) {
				sendByte(fromSw, toSw, sm, c.CONFIRM_COMM_ERROR)
				return
			}

			sendByte(fromSw, toSw, sm, c.CONFIRM_OK)
			log.Infof("Создан каталог %s.", oriFile)

		} else {
			if !saveReceivedFile(fileName, header, fileBody) {
				sendByte(fromSw, toSw, sm, c.CONFIRM_COMM_ERROR)
			}
			sendByte(fromSw, toSw, sm, c.CONFIRM_OK)
			log.Infof("Получен файл %s...", fileName)
		}

		SetNewWorkPath(conf.Conf.WorkPath())

	}

}

//
// Отсылает Ориону информацию по размерам
//
func opSendDirInfo(fromSr, fromSw, toSw chan SerialMessage, sm *m.StateMachine) {
	var inBytes uint16 = 0
	var inKBytes uint16 = 0

	subCommand, ok := receiveByte(fromSr, sm)
	if ok {
		switch subCommand {
		case c.OP_SEND_DIR_INFO_USED:
			inBytes = calcDirFileSize()
			inKBytes = inBytes >> 10
			log.Infof("Отправка занятого места на диске: %dK;", inKBytes)
		case c.OP_SEND_DIR_INFO_FREE:
			free := c.ORI_DIR_MAX_SIZE - uint32(calcDirFileSize())
			inBytes = uint16(free & 0x3FF)
			inKBytes = uint16(free >> 10)
			log.Infof("Отправка свободного места на диске: %dK;", inKBytes)
		case c.OP_SEND_DIR_INFO_SIZE:
			inBytes = 0
			inKBytes = c.ORI_DIR_MAX_SIZE_K
			log.Infof("Отправка полного размера диска: %dK;", inKBytes)
		}
		buff := []byte{byte(inBytes), byte(inBytes >> 8), byte(inKBytes), byte(inKBytes >> 8)}
		toSw <- BuildSerialMessage(SC_SEND_BUFF, 0, &buff)
		anywaySetOk(fromSw, sm)
	}
}

//
// Отсылает Ориону статус изменения каталога
//
func opSendDirStatus(fromSw, toSw chan SerialMessage, sm *m.StateMachine) {
	if dirChanged {
		sendByte(fromSw, toSw, sm, 1)
	} else {
		sendByte(fromSw, toSw, sm, 0)
	}
	sm.SetState(m.S_READY)
}

//
// Прием изменений (заголовка файла) каталога
//
func opReceiveChanges(fromSr, fromSw, toSw chan SerialMessage, sm *m.StateMachine) {
	fileNo, ok := receiveByte(fromSr, sm)
	if !ok {
		return
	}
	size := byte(len(*sentDirData) / 16)
	if fileNo > 0 && fileNo <= size {
		fileNo--
		if !sendByte(fromSw, toSw, sm, c.CONFIRM_DIR_OK) { // Запрос заголовка
			return
		}
		header := receiveHeader(fromSr, fromSw, toSw, sm)
		if header != nil {
			offset := uint16(fileNo) * 16
			dirEntry := (*sentDirData)[offset : offset+16]

			pos := -1
			for ctr := 0; ctr < 16; ctr++ {
				if (*header)[ctr] != dirEntry[ctr] {
					pos = ctr
					break
				}
			}

			if pos < 0 {
				log.Infof("Изменения в заголовке не обнаружены")
				if !sendByte(fromSw, toSw, sm, c.CONFIRM_OK) {
					return
				}
			} else {
				file := (*dirData)[fileNo].Name
				valid, isOri := isValidFile(file)
				if valid {
					fHeader, err := readHeader(file, isOri)
					if err != nil {
						sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
						return
					}
					if !isOri {
						(*fHeader)[0x0f] = 0
						(*fHeader)[0x0e] = 0
						(*fHeader)[0x0d] = 0
					}
					if bytes.Equal(*fHeader, dirEntry) {

						if pos == 0 && (*header)[0] == c.MARKER_DEL { // удаление файла
							deleteOriFile(fromSw, toSw, sm, file)
						} else if isOri {
							// работаем только с .ORI файлами
							if pos < 8 { // переименование файла
								for n := uint16(0); n < 8; n++ {
									b := (*header)[n]
									(*fHeader)[n] = b
									(*sentDirData)[offset+n] = b
								}
								renameOriFile(fromSw, toSw, sm, file, header)
							} else if pos == 8 || pos == 9 { // изменен адрес посадки файла
								(*fHeader)[8] = (*header)[8]
								(*fHeader)[9] = (*header)[9]
								log.Infof("Изменен адрес посадки файла %s на %X", file, uint16((*header)[8])+uint16((*header)[9])*256)
							} else if pos == 10 || pos == 11 { // изменение длины файла
								log.Warnf("Нельзя изменить длину файла %s", file)
								sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
								return
							} else if pos == 12 {
								(*fHeader)[12] = (*header)[12]
								log.Infof("Изменены атрибуты файла %s на %X", file, (*header)[12])
							} else if pos == 13 {
								(*fHeader)[13] = (*header)[13]
								log.Infof("Изменена страница в ОЗУ файла %s на %X", file, (*header)[13])
							} else {
								sendByte(fromSw, toSw, sm, c.CONFIRM_OTHER_ERROR)
								log.Infof("Изменена дата файла %s", file)
							}
							if !writeOriHeader(file, header) {
								sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
							}
							sendByte(fromSw, toSw, sm, c.CONFIRM_OK)
							dirData = ReadDir(conf.Conf.WorkPath())
						} else {
							sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
						}

					} else {
						log.Warnf("Заголовок в памяти и на диске, отличаются!")
						sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
					}

				} else {
					// не валидный файл
					sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
				}
			}

		}

	} else {
		log.Warnf("Номер файла больше чем количество файлов в каталоге!")
		sendByte(fromSw, toSw, sm, c.CONFIRM_TOO_MANY_FILES)
	}
}

//
// Проверяет файл на то, что у него есть заголовок и нв то, что это .ORI файл
//
func isValidFile(name string) (valid bool, isOri bool) {
	fullName := getFullName(name)

	f, err := os.Open(fullName)
	defer closeFile(f)
	if err != nil {
		log.Warnf("Не удалось открыть файл %s! Ошибка: %s", fullName, err.Error())
		return false, false
	}

	header := make([]byte, 16)
	_, err = f.Read(header)
	if err != nil {
		log.Warnf("Не удалось прочитать заголовок файла %s; Ошибка: %s!", fullName, err.Error())
		return false, false
	}

	ext := strings.ToUpper(filepath.Ext(name))
	if ext == c.EXT_ORI {
		if bytes.Equal(c.ORI_HEADER, header) {
			return true, true
		} else {
			log.Warnf("Файл %s с расширением .ORI не имеет сигнатуры!", fullName)
			return false, false
		}

	} else {
		return true, false
	}

}

//
// Расчитывает общий размер файлов в каталоге
//
func calcDirFileSize() uint16 {
	var size uint16 = 0
	if sentDirData == nil {
		if dirData == nil {
			dirData = ReadDir(conf.Conf.WorkPath())
		}
		sentDirData, _ = buildOriDir(dirData)
	}
	count := len(*sentDirData) / 16
	offset := 0
	for n := 0; n < count; n++ {
		dirItem := (*sentDirData)[offset : offset+16]
		size += uint16(dirItem[10]) + uint16(dirItem[11])*256
	}
	return size
}

//
// Возвращает полный путь к файлу
//
func getFullName(name string) string {
	dir := conf.Conf.WorkPath()
	if !strings.HasSuffix(dir, string(os.PathSeparator)) {
		dir += string(os.PathSeparator)
	}
	if strings.HasPrefix(name, ".") {
		name = name[1:]
	}
	return dir + name
}

func createDirectory(name string) bool {
	fullName := getFullName(strings.Trim(name, " "))
	err := os.Mkdir(fullName, 0770)
	if err != nil {
		log.Warnf("Не удалось создать каталог %s! Ошибка: %s", fullName, err.Error())
		return false
	}
	return true
}

func saveReceivedFile(name string, header *[]byte, fileBody *[]byte) bool {
	fullName := getFullName(name)
	f, err := os.OpenFile(fullName, os.O_CREATE|os.O_WRONLY, 0644)
	defer closeFile(f)
	if err != nil {
		log.Warnf("Не удалось создать файл %s! Ошибка: %s", fullName, err.Error())
		return false
	}
	_, err = f.Write(c.ORI_HEADER)
	if err != nil {
		log.Warnf("Не удалось записать сигнатуру .ORI в файл %s! Ошибка: %s", fullName, err.Error())
		return false
	}
	_, err = f.Write(*header)
	if err != nil {
		log.Warnf("Не удалось записать заголовок в файл %s! Ошибка: %s", fullName, err.Error())
		return false
	}
	_, err = f.Write(*fileBody)
	if err != nil {
		log.Warnf("Не удалось записать тело .ORI файла %s! Ошибка: %s", fullName, err.Error())
		return false
	}

	return true
}

func findFileInDir(header *[]byte) (byte, int) {

	fileName := (*header)[0:8]
	size := len(*sentDirData) / 16
	offset := 0
	for n := 0; n < size; n++ {
		fileRec := (*sentDirData)[offset : offset+8]
		if bytes.Equal(fileName, fileRec) {
			return byte(n), offset
		}
		offset += 16
	}
	return 0, -1
}

//
// Принимает 16-ти байтный заголовок от Ориона, проверяет контрольную сумму
//
func receiveHeader(fromSr, fromSw, toSw chan SerialMessage, sm *m.StateMachine) *[]byte {
	header, checkSum, ok := receiveBytes(fromSr, sm, t.FILE_HEADER_LEN)
	if !ok {
		return nil
	}
	oriCheckSum, ok := receiveByte(fromSr, sm)
	if !ok {
		return nil
	}
	if oriCheckSum != checkSum {
		sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
		log.Warnf("Ошибка контрольной суммы заголовка!")
		return nil
	} else {
		return header
	}
}

//
// Обрабатывает операции смены каталога
//
func changeDir(fromSr, fromSw, toSw chan SerialMessage, sm *m.StateMachine, name string, fileName []byte) {
	fullName := getFullName(name)
	stat, err := os.Stat(fullName)

	if err != nil || !stat.IsDir() {
		log.Warnf("Не найден каталог: %s", fullName)
		sendByte(fromSw, toSw, sm, c.CONFIRM_DIR_MISMATCH)
		return
	}
	if !sendByte(fromSw, toSw, sm, c.CONFIRM_OK) { // выдаём подтверждение о готовности файла каталога
		return
	}
	if !sendBytes(fromSw, toSw, sm, &[]byte{byte(len(fullName)), byte(len(fullName) >> 8)}) { // отправка длины
		return
	}
	confirm, ok := receiveByte(fromSr, sm)
	if !ok {
		return
	}
	if confirm != c.CONFIRM_SIZE_OK {
		log.Warnf("Несоответствие длины файла!")
		return
	}
	var fnBuff = []byte(fullName)
	fnBuff = append(fnBuff, checkSum(&fnBuff))
	if !sendBytes(fromSw, toSw, sm, &fnBuff) {
		return
	}

	confirm, ok = receiveByte(fromSr, sm)
	if !ok {
		return
	}
	if confirm == c.CONFIRM_COMM_ERROR {
		log.Warnf("Ошибка коммуникации!")
		return
	}
	if confirm == c.CONFIRM_ERROR_CS {
		log.Warnf("Ошибка контрольной суммы каталога!")
		return
	}

	// проверка на каталог .. (вверх)
	if fileName[1] == '.' && fileName[2] == ' ' {
		newDir := conf.Conf.WorkPath()
		if len(newDir) > 2 {
			if strings.HasSuffix(newDir, string(os.PathSeparator)) { // удалим последний слеш
				newDir = newDir[0 : len(newDir)-1]
			}
			pos := strings.LastIndex(newDir, string(os.PathSeparator))
			if pos > 1 {
				newDir = newDir[0:pos]
			}
		}
		fullName = newDir
	}

	SetNewWorkPath(fullName)

}

//
// Отправляет тело файла на Орион
//
func sendFileBody(fromSr, fromSw, toSw chan SerialMessage, sm *m.StateMachine, name string, fileName []byte, fileEntry []byte) {
	fullName := getFullName(name)
	// обычный файл
	_, err := readHeader(name, true)
	if err != nil { // не удалось получить заголовок файла
		sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
		return
	}

	// Если это DS-DOS .ORI файл, проверим его заголовок
	ext := strings.ToUpper(filepath.Ext(name))
	isOri := false
	if ext == c.EXT_ORI {
		isOri = isOriFile(name)
		if !isOri {
			log.Warnf("Файл %s имеет некорректный .ORI заголовок", fileName)
			sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
			return
		}
	}

	fileData := readFileBody(fullName, isOri)
	size := int(uint(fileEntry[10]) + uint(fileEntry[11])*256)

	if fileData == nil { // не удалось прочитать содержимое файла
		sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
		return
	}

	if size != len(*fileData) {
		log.Warnf("Отличается размер данных файла на диске %d и в заголовке %d!", len(*fileData), size)
		sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
		return
	}

	// Файл готов к передаче
	if !sendByte(fromSw, toSw, sm, c.CONFIRM_OK) {
		return
	}
	// Передаем размер файла
	if !sendBytes(fromSw, toSw, sm, &[]byte{fileEntry[10], fileEntry[11]}) {
		return
	}

	confirm, ok := receiveByte(fromSr, sm)
	if !ok {
		return
	}
	if confirm != c.CONFIRM_SIZE_OK {
		log.Warn("Несоответствие длины файла!")
		return
	}

	log.Infof("Отправка файла %s", fullName)
	if !sendBytes(fromSw, toSw, sm, fileData) {
		return
	}

	if !sendByte(fromSw, toSw, sm, checkSum(fileData)) {
		return
	}

	confirm, ok = receiveByte(fromSr, sm)
	if !ok {
		return
	}
	if confirm == c.CONFIRM_COMM_ERROR {
		log.Warnf("Ошибка коммуникации!")
		return
	}
	if confirm == c.CONFIRM_ERROR_CS {
		log.Warnf("Ошибка контрольной суммы файла!")
		return
	}
	log.Infof("Отправлен файл %s", fullName)
}

//
// Читает в буфер тело файла, пропуская заголовок
//
func readFileBody(fullName string, isOri bool) *[]byte {
	f, err := os.Open(fullName)
	defer closeFile(f)
	if err != nil {
		log.Warnf("Не удалось открыть файл %s Ошибка: %s", fullName, err.Error())
		return nil
	}
	stat, _ := f.Stat()

	if stat.Size() > 65520 || stat.Size() < 16 {
		log.Warnf("Файл %s имеет некорректный размер: %d байт!", fullName, stat.Size())
		return nil
	}

	headerSize := int64(16)
	if isOri {
		headerSize = 32
	}
	_, err = f.Seek(headerSize, 0)
	if err != nil {
		log.Warnf("Не удалось найти данные в файле %s Ошибка: %s", fullName, err.Error())
		return nil
	}
	buff := make([]byte, stat.Size()-headerSize)
	_, err = f.Read(buff)
	if err != nil {
		log.Warnf("Не удалось прочитать файл %s Ошибка: %s", fullName, err.Error())
		return nil
	}
	return &buff
}

func deleteOriFile(fromSw, toSw chan SerialMessage, sm *m.StateMachine, name string) bool {
	deleted := deleteFile(name)
	if !deleted {
		log.Warnf("Ошибка удаления файла %s", name)
		sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
	} else {
		log.Infof("Удален файл %s", name)
		sendByte(fromSw, toSw, sm, c.CONFIRM_OK)
		dirData = ReadDir(conf.Conf.WorkPath())
	}
	return deleted
}

func deleteFile(name string) bool {
	fullName := getFullName(name)
	err := os.Remove(fullName)
	if err != nil {
		log.Warnf("Не удалось удалить файл %s Ошибка: %s", fullName, err.Error())
		return false
	}
	return true
}

func renameOriFile(fromSw, toSw chan SerialMessage, sm *m.StateMachine, oldName string, newName *[]byte) {
	nName := extractFileName(newName)
	if nName != "" {
		if !renameFile(oldName, nName) {
			sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
			return
		}
	} else {
		log.Warnf("Пустое имя файла!")
		sendByte(fromSw, toSw, sm, c.CONFIRM_ERROR)
		return
	}
}

//
// Переименовывает старый файл в новый
//
func renameFile(oName, nName string) bool {
	fullOName := getFullName(oName)
	fullNName := getFullName(nName)
	err := os.Rename(fullOName, fullNName)
	if err != nil {
		log.Warnf("Не удалось переименовать файл %s в файл %s Ошибка: %s", fullOName, fullNName, err.Error())
		return false
	}
	return true
}

//
// Ищет расхождения между каталогами, возвращает false и номер файла, ели расхождения есть
//
func isEqualDirData(d1, d2 *[]byte) (bool, byte, byte) {
	offset := 0
	c1 := len(*d1) / 16
	c2 := len(*d2) / 16
	if c1 != c2 {
		log.Warnf("Несоответствие количества файлов в каталогах!")
		return false, 0, 0
	}

	for n := byte(0); n < byte(c1); n++ {
		for ptr := byte(0); ptr < 16; ptr++ {
			if (*d1)[offset] != (*d2)[offset] {
				return false, n, ptr
			}
		}
	}
	return true, 0, 0
}

//
// Добавляет каталог в список файлов
//
func appendDir(file os.FileInfo, dir *[]t.FileInfo) {
	if len(file.Name()) > 1 && !strings.HasPrefix(file.Name(), ".") {
		*dir = append(*dir, t.FileInfo{Name: file.Name(), Size: file.Size(), FType: c.FT_DIR})
	}
}

//
// Проверяет файл на принадлежность к поддерживаемым Орионовским файлам
// и добавляет его в список
//
func checkAndAppendFile(file os.FileInfo, dir *[]t.FileInfo) {
	name := file.Name()
	ext := strings.ToUpper(filepath.Ext(name))
	switch ext {
	case c.EXT_ORI:
		if isOriFile(name) {
			header, err := readHeader(name, true)
			if err == nil {
				addFile(dir, file, c.FT_FILE_ORI, header)
			}
		}
	case c.EXT_BRU:
		header, err := readHeader(name, false)
		if err == nil {
			addFile(dir, file, c.FT_FILE_BRU, header)
		}
	case c.EXT_ORD:
		header, err := readHeader(name, false)
		if err == nil {
			addFile(dir, file, c.FT_FILE_ORD, header)
		}
	}
}

//
// Добавляет файл ф каталог, анализируя заголовок
//
func addFile(dir *[]t.FileInfo, file os.FileInfo, ft int, header *[]byte) {
	ofh := t.Build(header)
	*dir = append(*dir, t.FileInfo{file.Name(), file.Size(), ofh.Addr(), ofh.Size(), c.FT_FILE_ORI})
}

//
// Возвращает заголовок файла, 16 байт
//
func readHeader(name string, shift bool) (*[]byte, error) {
	fullName := getFullName(name)
	file, err := os.Open(fullName)
	defer closeFile(file)

	if err != nil {
		log.Warnf("Не удалось открыть файл %s Ошибка: %s", fullName, err.Error())
		return nil, err
	}

	header := make([]byte, 16)
	if shift {
		_, err = file.Seek(16, 0)
		if err != nil {
			return nil, err
		}
	}
	size, err := file.Read(header)
	if err != nil {
		log.Warnf("Не удалось прочитать файл %s Ошибка: %s", fullName, err.Error())
		return nil, err
	}
	if size < 16 {
		return nil, errors.New("FILE_TOO_SHORT")
	}
	return &header, nil
}

//
// Записывает заголовок в .ORI файл
//
func writeOriHeader(name string, header *[]byte) bool {
	fullName := getFullName(name)
	file, err := os.OpenFile(fullName, os.O_WRONLY, 0644)
	defer closeFile(file)

	if err != nil {
		log.Warnf("Не удалось открыть файл %s; Ошибка: %s", fullName, err.Error())
		return false
	}

	_, err = file.Seek(16, 0)
	if err != nil {
		log.Warnf("Не удалось найти заголовок в файле %s; Ошибка: %s", fullName, err.Error())
		return false
	}

	_, err = file.Write(*header)
	if err != nil {
		log.Warnf("Не удалось записать заголовок в файл %s; Ошибка: %s", fullName, err.Error())
		return false
	}
	return true

}

//
// Извлекает имя файла из заголовка
//
func extractFileName(header *[]byte) string {
	res := ""
	for i := 0; i < 8; i++ {
		b := (*header)[i]
		if b < 0x20 {
			res += "_"
		} else if b == 0x20 {
			break
		} else {
			res += string(b)
		}
	}
	return res
}

//
// Проверяет сигнатуру .ORI файла
//
func isOriFile(name string) bool {
	header, err := readHeader(name, false)
	if err != nil {
		return false
	}
	isEqual := bytes.Equal(*header, c.ORI_HEADER)
	if !isEqual {
		log.Warnf("Сигнатура не соотверствует файлу .ORI: %s", name)
	}
	return isEqual
}

func isBruFile(name string) bool {
	return true
}

func isOrdFile(name string) bool {
	return true
}

//
// Возвращает слайс 16-байт в формате элемента каталога Ориона
//
func getOriDirItem(name string) []byte {
	var res []byte
	if !strings.HasPrefix(name, ".") {
		res = append(res, '.')
	}

	chars := encodeToKoi7(strings.ToUpper(name))

	for n, ch := range chars {
		if n == 7 {
			break
		} else if ch < 0x20 {
			res = append(res, '_')
		} else if ch > 0x40 {
			res = append(res, byte(ch)&0x5f)
		} else {
			res = append(res, byte(ch))
		}
	}
	for len(res) < 8 {
		res = append(res, ' ')
	}

	length := len(getFullName(name))
	return append(res, 0, 0, byte(length), byte(length>>8), 0, 0, 0, 0)

}

func closeFile(f *os.File) {
	err := f.Close()
	if err != nil {
		log.Warnf("Не удалось закрыть файл: %s; Ошибка: %s", f.Name(), err.Error())
	}
}

func encodeToKoi7(str string) []uint8 {
	var res []byte
	for _, ch := range strings.ToUpper(str) {
		enc, exists := c.ENCODE_TO_KOI7[ch]
		if !exists {
			enc = ch
		}
		res = append(res, byte(enc))
	}
	return res
}

func checkSum(buff *[]byte) byte {
	var checkSum byte = 0
	for _, b := range *buff {
		checkSum ^= b
	}
	return checkSum
}
