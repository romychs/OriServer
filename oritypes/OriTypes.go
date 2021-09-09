package oritypes

import "fmt"

type KeyValue struct {
	Value string
	Key   int
}

type KeyValueArray []KeyValue

type FileInfo struct {
	Name  string
	Size  int64
	Addr  uint16
	Len   uint16
	FType byte
}

//
// Структура заголовка файла Ориона
//
type OrionFileHeader struct {
	name [8]byte // Имя файла
	addr uint16  // Адрес посадки в ОЗУ
	size uint16  // Размер
	attr byte    // Аттрибуты
	page byte    // Страница в ОЗУ
	date uint16  // Дата
}

const DIR_ENTRY_LEN = 16
const FILE_HEADER_LEN = 16

type OrionDirectiry []OrionFileHeader

func Build(buff *[]byte) *OrionFileHeader {
	var h OrionFileHeader
	copy(h.name[:], (*buff)[0:8])
	h.addr = uint16((*buff)[8]) + (uint16((*buff)[9]) << 8)
	h.size = uint16((*buff)[10]) + (uint16((*buff)[11]) << 8)
	h.attr = (*buff)[12]
	h.page = (*buff)[13]
	h.date = uint16((*buff)[14]) + (uint16((*buff)[15]) << 8)
	return &h
}

func (h *OrionFileHeader) Name() []byte {
	return h.name[:]
}

func (h *OrionFileHeader) Addr() uint16 {
	return h.addr
}

func (h *OrionFileHeader) AddrLo() byte {
	return byte(h.addr)
}

// Возвращает старший байт адреса посадки
func (h *OrionFileHeader) AddrHi() byte {
	return byte(h.addr >> 8)
}

// Возвращает размер файла
func (h *OrionFileHeader) Size() uint16 {
	return h.size
}

// Возвращает младший байт размера файла
func (h *OrionFileHeader) SizeLo() byte {
	return byte(h.size)
}

// Возвращает старший байт размера файла
func (h *OrionFileHeader) SizeHi() byte {
	return byte(h.size >> 8)
}

// Возвращает атрибуты файла
func (h *OrionFileHeader) Attr() byte {
	return h.attr
}

// Возвращает страницу ОЗУ, посадки файла
func (h *OrionFileHeader) Page() byte {
	return h.page
}

// Возвращает дату файла
func (h *OrionFileHeader) Date() uint16 {
	return h.date
}

// Возвращает дату файла в виде строки DD.MM.YYYY
func (h *OrionFileHeader) DateStr() string {
	y := h.date & 0x7F
	if y >= 99 {
		y = 1999
	} else {
		y += 2000
	}
	m := (h.date>>7)&0x0F + 1
	d := (h.date>>11)&0x1F + 1
	return fmt.Sprintf("%02d.%02d.%4d", d, m, y)
}

// Возвращает заголовок в виде слайса
func (h *OrionFileHeader) Bytes() []byte {
	res := h.name[:]
	res = append(res, byte(h.addr), byte(h.addr>>8))
	res = append(res, byte(h.size), byte(h.size>>8))
	res = append(res, h.attr, h.page)
	res = append(res, byte(h.date), byte(h.date>>8))
	return res
}
