package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/user"

	log "github.com/sirupsen/logrus"
	"ru.romych/OriServer/constants"
)

type Config struct {
	SerialConfig SerialConf `json:"serial"`
	Path         string     `json:"workPath"` // /home/user/Orion
}

type SerialConf struct {
	ComPort     string `json:"comPort"`     // /dev/ttyUSB0
	ComSpeed    int    `json:"comSpeed"`    // 38400
	ComDataBits int    `json:"comDataBits"` // 8
	ComOdd      int    `json:"comOdd"`      // 1-Odd/2-Even/0-None
	ComStopBits int    `json:"comStopBits"` // 1

}

func init() {
	log.Infof("Загрузка файла конфигурации %s", constants.CONFIG_FILE)
	var err error
	Conf, err = LoadConfig(constants.CONFIG_FILE)
	if err != nil {
		log.Fatal("Не удалось загрузить файл конфигурации!", err)
	}
}

var Conf *Config = nil

func LoadConfig(fileName string) (*Config, error) {

	bytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		bytes, err = ioutil.ReadFile("./" + fileName)
		if err != nil {
			return &Config{}, err
		}
	}

	var config Config
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return &Config{}, err
	}

	if config.SerialConfig.ComSpeed < 1 {
		config.SerialConfig.ComSpeed = 38400
	}

	// Проверка каталога на существование
	if len(config.Path) > 0 {
		s, err := os.Stat(config.Path)
		if err != nil || !s.IsDir() {
			log.Warnf("Не удалось получить информацию по каталогу: \"%s\", будет выбран коталог по умолчанию.", config.Path)
			config.Path = ""
		}
	}
	// Если путь не указан или не существует, выберем домашний каталог пользователя
	if len(config.Path) == 0 {
		usr, err := user.Current()
		if err != nil {
			log.Fatal(err)
		}
		config.Path = usr.HomeDir
	}

	return &config, nil
}

func (s *SerialConf) Equals(other SerialConf) bool {
	return s.ComSpeed == other.ComSpeed && s.ComPort == other.ComPort &&
		s.ComStopBits == other.ComStopBits && s.ComDataBits == other.ComDataBits &&
		s.ComOdd == other.ComOdd
}

func (c *Config) SetWorkPath(path string) {
	c.Path = path
}

func (c *Config) WorkPath() string {
	return c.Path
}

func (c *Config) SaveDefaultConfig() {
	c.SaveConfig(constants.CONFIG_FILE)
}

func (c *Config) SaveConfig(fileName string) {
	bytes, err := json.MarshalIndent(c, "", "   ")
	if err != nil {
		log.Warnf("Не удалось преобразовать конфиг в Json: %s", err.Error())
		return
	}
	err = ioutil.WriteFile(fileName, bytes, 0664)
	if err != nil {
		log.Warnf("Не удалось сохранить конфиг в файл %s, ошибка: %s", fileName, err.Error())
	}
}

//
// Возвращает параметры настройки порта в виде строки
//
// func (c *Config) GetSerialStr() string {
// 	// /dev/ttyS0,9600,8,N,1
// 	//ev,ok := EVEN_VALUES[c.SerialConfig.ComOdd]
// 	return fmt.Sprintf("%s,%d,%d,%s,%d", c.SerialConfig.ComPort,c.SerialConfig.ComSpeed, c.SerialConfig.ComDataBits, "N", c.SerialConfig.ComStopBits )
// }
