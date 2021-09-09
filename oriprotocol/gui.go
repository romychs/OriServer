package oriprotocol

import (
	"fmt"
	"github.com/romychs/gotk3/glib"
	"github.com/romychs/gotk3/gtk"
	log "github.com/sirupsen/logrus"
	conf "ru.romych/OriServer/config"
	c "ru.romych/OriServer/constants"
	t "ru.romych/OriServer/oritypes"
)

//var listStore *gtk.ListStore

const (
	COL_NAME = iota
	COL_SIZE
	COL_TYPE
	COL_ADDR
	COL_LENGTH
)

const (
	ALIGN_LEFT   = 0.0
	ALIGN_RIGHT  = 1.0
	ALIGN_CENTER = 0.5
)

const (
	MSG_CANT_CREATE = "Не удалось инициализировать UI: "
	MSG_CANT_UPDATE = "Не удалось обновить UI: "
)

var (
	configDialogBuilder *gtk.Builder
	mainWindowBuilder   *gtk.Builder
	mainWindow          *gtk.Window
	filesView           *gtk.TreeView
	configDialog        *gtk.Dialog
	buttonConnect       *gtk.ToggleButton
	labelWorkPath       *gtk.Label
	labelWorkInfo       *gtk.Label
	labelComInfo        *gtk.Label
)

//
// Инициализация основного окна приложения
//
func InitMainWindow() {

	// Создаём билдер
	var err error
	mainWindowBuilder, err = gtk.BuilderNew()
	if err != nil {
		log.Fatal("Ошибка:", err)
	}

	// Загружаем в билдер окно из файла Glade
	err = mainWindowBuilder.AddFromFile("ui/OriServer.glade")
	if err != nil {
		log.Fatal("Ошибка:", err)
	}

	filesView = initFilesView()

	configDialog = initConfigDialog()

	// Обработчики событий

	// События выхода из приложения
	mainWindow = getObjectById(mainWindowBuilder, "top_window").(*gtk.Window)
	_, err = mainWindow.Connect("destroy", exitApp)
	if err != nil {
		log.Fatal(MSG_CANT_CREATE, err)
	}

	labelWorkPath = getLabelById(mainWindowBuilder, "l_work_path")
	labelWorkInfo = getLabelById(mainWindowBuilder, "l_work_info")
	labelComInfo = getLabelById(mainWindowBuilder, "l_com_info")

	labelWorkPath.SetLabel(conf.Conf.WorkPath())
	updateLabelComInfo(&conf.Conf.SerialConfig)

	btExit := getButtonById(mainWindowBuilder, "btn_exit")
	_, err = btExit.Connect("clicked", exitApp)
	if err != nil {
		log.Fatal(MSG_CANT_CREATE, err)
	}

	miExit := getMenuItemById(mainWindowBuilder, "menu_item_exit")
	_, err = miExit.Connect("activate", exitApp)
	if err != nil {
		log.Fatal(MSG_CANT_CREATE, err)
	}

	// Кнопка установки соединения
	buttonConnect = getObjectById(mainWindowBuilder, "btn_connect").(*gtk.ToggleButton)
	_, err = buttonConnect.Connect("toggled", func() {
		if buttonConnect.GetActive() {
			StartSerialHandlers()
		} else {
			DisconnectSerial()
		}
	})
	if err != nil {
		log.Fatal(MSG_CANT_CREATE, err)
	}

	buttonChoiceDir := getObjectById(mainWindowBuilder, "btn_choice_dir").(*gtk.Button)
	_, err = buttonChoiceDir.Connect("clicked", runChoiceDirDialog)

	// Событие выбора пункта в списке файлов
	sel, _ := filesView.GetSelection()
	sel.SetMode(gtk.SELECTION_SINGLE)
	_, err = sel.Connect("changed", selectionChanged)
	if err != nil {
		log.Fatal(MSG_CANT_CREATE, err)
	}

	// События нажатия кнопки конфигукации
	btConf := getButtonById(mainWindowBuilder, "btn_conf")
	_, err = btConf.Connect("clicked", runConfigDialog)
	if err != nil {
		log.Fatal(MSG_CANT_CREATE, err)
	}

	buttonAbout := getObjectById(mainWindowBuilder, "btn_about").(*gtk.Button)
	_, err = buttonAbout.Connect("clicked", func() {
		dialog := initAboutDialog()
		dialog.Run()
		dialog.Destroy()
	})
	if err != nil {
		log.Fatal(MSG_CANT_CREATE, err)
	}

	mainWindow.ShowAll()
}

func initFilesView() *gtk.TreeView {
	fv := getObjectById(mainWindowBuilder, "tw_files").(*gtk.TreeView)
	fv.AppendColumn(createColumn("Название", 0))
	fv.AppendColumn(createColumn("Размер", 1))
	fv.AppendColumn(createColumn("Тип", 2))
	fv.AppendColumn(createColumn("Адрес", 3))
	fv.AppendColumn(createColumn("Длина", 4))
	fv.SetModel(buildFileListModel())
	return fv
}

//
// Вызов диалога доля выбора рабочего каталога
//
func runChoiceDirDialog() {

	dialog := initDirChoiceDialog(conf.Conf.WorkPath())

	if dialog.Run() == gtk.RESPONSE_ACCEPT {
		path := dialog.GetFilename()
		log.Debugf("Selected path: %s;", path)
		SetNewWorkPath(path)
	}
	dialog.Destroy()
}

//
// Возвращает хранилище со списком файлов
//
func buildFileListModel() *gtk.ListStore {
	dirData = ReadDir(conf.Conf.WorkPath())

	listStore, err := gtk.ListStoreNew(glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING, glib.TYPE_STRING)
	if err != nil {
		log.Fatal("Не удалось создать хранилище списка файлов:", err)
	}

	for _, i := range *dirData {
		addRow(listStore, i)
	}

	return listStore
}

//
// Инициалтзация диалога выбора каталога
//
func initDirChoiceDialog(path string) *gtk.FileChooserDialog {
	dialog, err := gtk.FileChooserDialogNewWith2Buttons("Выбор рабочего каталога", mainWindow,
		gtk.FILE_CHOOSER_ACTION_SELECT_FOLDER,
		"Отмена", gtk.RESPONSE_CANCEL,
		"Выбрать", gtk.RESPONSE_ACCEPT,
	)
	if err != nil {
		log.Fatal(MSG_CANT_CREATE, err)
	}
	dialog.SetModal(true)
	dialog.SetFilename(path)
	return dialog

}

func initAboutDialog() *gtk.AboutDialog {
	image := getObjectById(mainWindowBuilder, "im_orion").(*gtk.Image)
	dialog, err := gtk.AboutDialogNew()
	if err != nil {
		log.Fatal(MSG_CANT_CREATE, err)
	}
	dialog.SetAuthors([]string{"Romych - Разработка.", "Denn - Оригинальная идея."})
	dialog.SetProgramName("Orion Server")
	dialog.SetComments("Приложение для взаимодействия c ПК Орион/Орион ПРО\nчерез последовательный порт.")
	dialog.SetLogo(image.GetPixbuf())
	dialog.SetVersion("Версия: " + c.APP_VERSION)

	return dialog
}

//
// Инициализация диалога конфигурации
//
func initConfigDialog() *gtk.Dialog {
	var err error
	configDialogBuilder, err = gtk.BuilderNew()
	if err != nil {
		log.Fatal(MSG_CANT_CREATE, err)
	}
	err = configDialogBuilder.AddFromFile("ui/Config.glade")
	if err != nil {
		log.Fatal(MSG_CANT_CREATE, err)
	}

	// Получаем объект главного окна по ID
	obj, err := configDialogBuilder.GetObject("dlg_conf")
	if err != nil {
		log.Fatal(MSG_CANT_CREATE, err)
	}
	dialog := obj.(*gtk.Dialog)

	btCancel := getButtonById(configDialogBuilder, "btn_cancel")
	_, err = btCancel.Connect("clicked", func() {
		dialog.Response(gtk.RESPONSE_CANCEL)
		dialog.Hide()
	})
	if err != nil {
		log.Fatal(MSG_CANT_CREATE, err)
	}

	// Обработчики событий кнопок
	btOk := getButtonById(configDialogBuilder, "btn_ok")
	_, err = btOk.Connect("clicked", func() {
		dialog.Response(gtk.RESPONSE_OK)
		dialog.Hide()
	})
	if err != nil {
		log.Fatal(MSG_CANT_CREATE, err)
	}

	dialog.SetModal(true)
	return dialog
}

//
// Отображение диалога конфигурации и обработка изменений
//
func runConfigDialog() {
	updateConfigDialog(configDialogBuilder)
	if configDialog.Run() == gtk.RESPONSE_OK {
		nc := conf.SerialConf{ComPort: getComboStringValue("cb_com_port"),
			ComSpeed:    getComboKey("cb_com_speed"),
			ComDataBits: getComboKey("cb_com_bits"),
			ComOdd:      getComboKey("cb_com_even"),
			ComStopBits: getComboKey("cb_com_stop"),
		}
		if !nc.Equals(conf.Conf.SerialConfig) {
			// Есть изменения в настройках порта
			if buttonConnect.GetActive() {
				DisconnectSerial()
				buttonConnect.SetActive(false)
			}
			conf.Conf.SerialConfig = nc
			updateLabelComInfo(&nc)
			log.Debugf("Изменения в параметрах соединения применены. %+v", conf.Conf.SerialConfig)
			conf.Conf.SaveDefaultConfig()
		} else {
			log.Debug("Нет изменений в параметрах соединения.")
		}
	}
}

//
// Отображает на метке в таскбаре конфигурацию соединения
//
func updateLabelComInfo(conf *conf.SerialConf) {
	ev := EVEN_VALUES[conf.ComOdd]
	comStr := fmt.Sprintf("%s,%d,%d%s%d", conf.ComPort, conf.ComSpeed, conf.ComDataBits, ev.Value[0:1], conf.ComStopBits)
	labelComInfo.SetLabel(comStr)
}

//
// Отображает диалог с сообщением об ошибке
//
func showError(message string) {
	dialog, err := gtk.MessageDialogNew(mainWindow, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, &message)
	if err != nil {
		log.Warn("Не удалось создать диалог с ошибкой: ", err)
	}
	dialog.Run()
	dialog.Destroy()
}

//
// Отображает сообщение об ошибке соединения
//
func ShowConnectError(err error) {
	message := fmt.Sprintf("Ошибка соединения, порт: %s!\nОшибка: %s", conf.Conf.SerialConfig.ComPort, err.Error())
	_, e := glib.IdleAdd(showError, message)
	if e != nil {
		log.Warn("Не удалось отобразить диалог: ", e)
	}
	_, e = glib.IdleAdd(buttonConnect.SetActive, false)
	if e != nil {
		log.Warn("Не удалось деактивировать кнопку: ", e)
	}
}

//
// Задает текст указанной метке. Безопасно вызывать из горутин
//
func SetLabel(label *gtk.Label, value string) {
	_, e := glib.IdleAdd(label.SetLabel, value)
	if e != nil {
		log.Warn("Не удалось установить текст метки: ", e)
	}
}

//
// Устанавливает значение метки, отображающей рабочий каталог. Безопасно вызывать из горутин
//
func SetNewWorkPath(path string) {
	if path != conf.Conf.WorkPath() {
		conf.Conf.SetWorkPath(path)
		SetLabel(labelWorkPath, path)
		conf.Conf.SaveDefaultConfig()
	}
	_, e := glib.IdleAdd(filesView.SetModel, buildFileListModel())
	if e != nil {
		log.Warn("Не удалось в фоне установить модель со списком файлов", e)
	}
}

func SetWorkInfo(message string) {
	SetLabel(labelWorkInfo, message)
}

var firstComboInit = true // Флаг первой инициализации полей ComboBox

func checkUpdError(err error) {
	if err != nil {
		log.Fatal(MSG_CANT_UPDATE, err)
	}
}

//
// Обновляет значения полей диалога конфигурации на основе текущих значений
//
func updateConfigDialog(b *gtk.Builder) {
	cbComPort := getComboBoxById(b, "cb_com_port")
	UpdateNameValueCombo(cbComPort, ScanSerialPorts(), 0, conf.Conf.SerialConfig.ComPort)
	initCombo(cbComPort)

	cbComSpeed := getComboBoxById(b, "cb_com_speed")
	UpdateNameValueCombo(cbComSpeed, &BAUD_RATES, conf.Conf.SerialConfig.ComSpeed)
	initCombo(cbComSpeed)

	cbComBits := getComboBoxById(b, "cb_com_bits")
	UpdateNameValueCombo(cbComBits, &DATA_BITS, conf.Conf.SerialConfig.ComDataBits)
	initCombo(cbComBits)
	cbComBits.SetSensitive(false)

	cbComEven := getComboBoxById(b, "cb_com_even")
	UpdateNameValueCombo(cbComEven, &EVEN_VALUES, conf.Conf.SerialConfig.ComOdd)
	initCombo(cbComEven)
	cbComEven.SetSensitive(false)

	cbComStop := getComboBoxById(b, "cb_com_stop")
	UpdateNameValueCombo(cbComStop, &STOP_BITS, conf.Conf.SerialConfig.ComStopBits)
	initCombo(cbComStop)
	cbComStop.SetSensitive(false)
	firstComboInit = false
}

//
// Задает основные параметры ComboBox
//
func initCombo(cb *gtk.ComboBox) {
	if firstComboInit {
		cell, err := gtk.CellRendererTextNew()
		if err != nil {
			log.Warn(MSG_CANT_UPDATE, err)
			return
		}
		cell.SetAlignment(ALIGN_LEFT, ALIGN_CENTER)
		cb.PackStart(cell, false)
		cb.AddAttribute(cell, "text", 0)
	}
}

//
// Обновляет хранилище ComboBox значениями keyValues и выбирает пункт с ключом selectedKey
// или значением selectedValue, если указано
//
func UpdateNameValueCombo(cb *gtk.ComboBox, keyValues *t.KeyValueArray, selectedKey int, selectedValue ...string) {
	ls, err := gtk.ListStoreNew(glib.TYPE_STRING, glib.TYPE_INT)
	if err != nil {
		log.Warn(MSG_CANT_UPDATE, err)
		return
	}
	active := 0
	for n, item := range *keyValues {
		_, err = appendValues(ls, item.Value, item.Key)
		if err != nil {
			log.Warn(MSG_CANT_UPDATE, err)
			return
		}
		if len(selectedValue) > 0 {
			if item.Value == selectedValue[0] {
				active = n
			}
		} else {
			if item.Key == selectedKey {
				active = n
			}
		}
	}
	cb.SetModel(ls)
	cb.SetActive(active)
}

//
// Обработка сигнала выхода
//
func exitApp() {
	StopSerialHandlers()
	gtk.MainQuit()
}

//
// Обработка сигнала выбора пункта в списке файлов
//
func selectionChanged(s *gtk.TreeSelection) {
	model, e := filesView.GetModel()
	if e == nil && model != nil {
		_, i, ok := s.GetSelected()
		if ok {
			value, _ := model.GetValue(i, 0)
			if value != nil {
				st, e := value.GetString()
				if e == nil {
					log.Println(st)
				}
			}

		}
	}
}

//
// Возвращает метку по ее идентификатору
//
func getLabelById(b *gtk.Builder, id string) *gtk.Label {
	return getObjectById(b, id).(*gtk.Label)
}

//
// Возвращает кнопку по ее идентификатору
//
func getButtonById(b *gtk.Builder, id string) *gtk.Button {
	return getObjectById(b, id).(*gtk.Button)
}

//
// Возвращает пункт меню по его идентификатору
//
func getMenuItemById(b *gtk.Builder, id string) *gtk.MenuItem {
	return getObjectById(b, id).(*gtk.MenuItem)
}

//
// Возвращает ComboBox по ее идентификатору
//
func getComboBoxById(b *gtk.Builder, id string) *gtk.ComboBox {
	return getObjectById(b, id).(*gtk.ComboBox)
}

//
// Возвращает GLIB-объект по идентификатору
//
func getObjectById(b *gtk.Builder, id string) glib.IObject {
	obj, err := b.GetObject(id)
	if err != nil {
		log.Fatal("Ошибка:", err)
	}
	if obj == nil {
		log.Fatal("Не найден glib.IObject: ", id)
	}
	return obj
}

//
// Создает колонку для таблицы файлов
//
func createColumn(title string, id int) *gtk.TreeViewColumn {
	cellRenderer, err := gtk.CellRendererTextNew()
	if err != nil {
		log.Fatal("Unable to create text cell renderer:", err)
	}

	switch id {
	case COL_NAME:
		cellRenderer.SetAlignment(ALIGN_LEFT, ALIGN_CENTER)
		err = cellRenderer.SetProperty("ellipsize", 3)
		if err != nil {
			log.Warn("Can not set ellipsize:", err)
		}
	case COL_SIZE:
		cellRenderer.SetAlignment(ALIGN_RIGHT, ALIGN_CENTER)
	case COL_TYPE:
		cellRenderer.SetAlignment(ALIGN_CENTER, ALIGN_CENTER)
	default:
		cellRenderer.SetAlignment(ALIGN_RIGHT, ALIGN_CENTER)
	}
	column, err := gtk.TreeViewColumnNewWithAttribute(title, cellRenderer, "text", id)
	if err != nil {
		log.Fatal("Unable to create cell column:", err)
	}
	if id == COL_NAME {
		column.SetMinWidth(250)
	}
	column.SetSpacing(2)
	return column
}

var columnEnum = []int{COL_NAME, COL_SIZE, COL_TYPE, COL_ADDR, COL_LENGTH}

//
// Добавляет строку в список файлов
//
func addRow(listStore *gtk.ListStore, fileInfo t.FileInfo) {
	iter := listStore.Append()
	fileType := c.FILE_TYPE_NAME[fileInfo.FType]
	fSize := ""
	fAddr := ""
	fLen := ""
	if fileInfo.FType != c.FT_DIR {
		fSize = fmt.Sprintf("%d", fileInfo.Size)
		fAddr = fmt.Sprintf("%04X", fileInfo.Addr)
		fLen = fmt.Sprintf("%04X", fileInfo.Len)
	}
	err := listStore.Set(iter, columnEnum, []interface{}{fileInfo.Name, fSize, fileType, fAddr, fLen})
	if err != nil {
		log.Fatal("Unable to add row:", err)
	}
}

//
// Получние выбранного в ComboBox значения в виде строки
//
func getComboStringValue(cbName string) string {
	cb := getComboBoxById(configDialogBuilder, cbName)
	value, err := getComboValue(cb, 0)
	if err == nil {
		asString, err := value.GetString()
		if err != nil {
			log.Warn("Не удалось получить value.GetString(): ", err)
		} else {
			return asString
		}
	} else {
		log.Warn("Не удалось получить getComboValue(): ", err)
	}
	return ""
}

//
// Получние выбранного в ComboBox значения ключа
//
func getComboKey(cbName string) int {
	cb := getComboBoxById(configDialogBuilder, cbName)
	value, err := getComboValue(cb, 1)
	if err == nil {
		return value.GetInt()
	} else {
		log.Warn("Не удалось получить getComboKey(): ", err)
	}
	return -1
}

//
// Возвращает выбранное значение в ComboValue, либо nil, если не удалось
//
func getComboValue(cb *gtk.ComboBox, column int) (*glib.Value, error) {
	ti, err := cb.GetActiveIter()
	if err != nil {
		return nil, err
	}
	tm, err := cb.GetModel()
	if err != nil {
		return nil, err
	}
	val, err := tm.GetValue(ti, column)
	if err != nil {
		return nil, err
	}
	//	columnType := tm.GetColumnType(column)
	//	log.Debug("ColumnType:", columnType.Name())
	return val, nil
}

//
// Добавляет несколько значений к rowStore
//
func appendValues(ls *gtk.ListStore, values ...interface{}) (*gtk.TreeIter, error) {
	iter := ls.Append()
	for i := 0; i < len(values); i++ {
		err := ls.SetValue(iter, i, values[i])
		if err != nil {
			return nil, err
		}
	}
	return iter, nil
}
