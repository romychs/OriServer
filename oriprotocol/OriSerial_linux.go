// +build linux

package oriprotocol

import (
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"os"
	ot "ru.romych/OriServer/oritypes"
	"sort"
	"strings"
	"syscall"
	"unsafe"
)

var termios = unix.Termios{Oflag: 0, Cflag: 0}

//
// Получение списка последовательных портов
//
func ScanSerialPorts() *ot.KeyValueArray {
	var res ot.KeyValueArray
	files, err := ioutil.ReadDir("/dev/")
	if err != nil {
		log.Fatal("Не удалось получить список портов!", err)
	}
	no := 0
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "ttyS") || strings.HasPrefix(file.Name(), "ttyUSB") || strings.HasPrefix(file.Name(), "ttyACM") {
			name := "/dev/" + file.Name()
			// Сначала, откроем порт как файл с параметрами последовательного устройства
			f, err := os.OpenFile(name, unix.O_RDWR|unix.O_NOCTTY|unix.O_NONBLOCK, 0666)
			if err == nil {
				fd := f.Fd()
				// Прочитаем текущие параметры порта
				_, _, errno := unix.Syscall6(
					unix.SYS_IOCTL,
					fd,
					uintptr(unix.TIOCGSERIAL),
					uintptr(unsafe.Pointer(&termios)),
					0,
					0,
					0,
				)
				if f != nil {
					_ = f.Close()
				}
				if errno == syscall.Errno(0) || errno == syscall.ENOTTY {
					res = append(res, ot.KeyValue{Key: no, Value: name})
					log.Tracef("Port: %s\t\tOSpeed: %X; ISpeed: %X;", name, termios.Ospeed, termios.Ispeed)
					no++
				} else {
					log.Warnf("Ошибка вызова Syscall6: %d; для файла: %s", errno, name)
				}
			}
		}
	}

	sort.SliceStable(res, func(i, j int) bool {
		if len(res[i].Value) < len(res[j].Value) {
			return true
		}
		if strings.HasPrefix(res[i].Value, "/dev/ttyU") && !strings.HasPrefix(res[j].Value, "/dev/ttyU") {
			return true
		}
		return res[i].Value < res[j].Value
	})

	return &res
}
