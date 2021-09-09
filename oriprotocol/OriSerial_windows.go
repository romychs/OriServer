// +build windows

package oriprotocol

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
	ot "ru.romych/OriServer/oritypes"
	"sync"
	"syscall"
	"unsafe"
)

//
// Получение списка последовательных портов
//
func ScanSerialPorts() *ot.KeyValueArray {

	list, e := nativeGetPortsList()
	if e != nil {
		log.Warn("Ошибка", e)
	}
	log.Infof("Размер списка портов: %d", len(list))
	no := 0
	var res ot.KeyValueArray
	if len(list) == 0 {
		res = append(res, ot.KeyValue{
			Key:   no,
			Value: "не обнаружен",
		})
	}
	for p := range list {
		log.Infof("Port: %d", p)
		res = append(res, ot.KeyValue{
			Key:   no,
			Value: fmt.Sprintf("COM%d", p),
		})
		no++
	}
	return &res
}

type windowsPort struct {
	mu     sync.Mutex
	handle syscall.Handle
}

const errorGetPortList = "ошибка получения списка портов"

func nativeGetPortsList() ([]string, error) {
	subKey, err := syscall.UTF16PtrFromString("HARDWARE\\DEVICEMAP\\SERIALCOMM\\")
	if err != nil {
		return nil, errors.New(errorGetPortList)
	}

	var h syscall.Handle
	if err := syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, subKey, 0, syscall.KEY_READ, &h); err != nil {
		if errno, isErrno := err.(syscall.Errno); isErrno && errno == syscall.ERROR_FILE_NOT_FOUND {
			return []string{}, nil
		}
		return nil, errors.New(errorGetPortList)
	}
	defer syscall.RegCloseKey(h)

	var valuesCount uint32
	if syscall.RegQueryInfoKey(h, nil, nil, nil, nil, nil, nil, &valuesCount, nil, nil, nil, nil) != nil {
		return nil, errors.New(errorGetPortList)
	}

	list := make([]string, valuesCount)
	for i := range list {
		var data [1024]uint16
		dataSize := uint32(len(data))
		var name [1024]uint16
		nameSize := uint32(len(name))
		if regEnumValue(h, uint32(i), &name[0], &nameSize, nil, nil, &data[0], &dataSize) != nil {
			return nil, errors.New(errorGetPortList)
		}
		list[i] = syscall.UTF16ToString(data[:])
	}
	return list, nil
}

var (
	modadvapi32       = windows.NewLazySystemDLL("advapi32.dll")
	procRegEnumValueW = modadvapi32.NewProc("RegEnumValueW")
)

func regEnumValue(key syscall.Handle, index uint32, name *uint16, nameLen *uint32, reserved *uint32, class *uint16, value *uint16, valueLen *uint32) (regerrno error) {
	r0, _, _ := syscall.Syscall9(procRegEnumValueW.Addr(), 8, uintptr(key), uintptr(index), uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(nameLen)), uintptr(unsafe.Pointer(reserved)), uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(value)), uintptr(unsafe.Pointer(valueLen)), 0)
	if r0 != 0 {
		regerrno = syscall.Errno(r0)
	}
	return
}
