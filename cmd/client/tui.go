package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/victor/gophkeeper/internal/client/api"
	"github.com/victor/gophkeeper/internal/crypto"
	"github.com/victor/gophkeeper/internal/model"
)

// tuiState — состояние TUI-приложения.
type tuiState int

const (
	stateAuth tuiState = iota
	stateMain
	stateList
	stateStats
	stateCreate
	stateEdit
	stateConfirmDelete
	stateDownload
	statePassword
)

// mainMenuItems — пункты главного меню.
var mainMenuItems = []string{"Создать", "Статистика", "Выход"}

// objectItem — элемент списка объектов.
type objectItem struct {
	obj *model.Object
}

func (i objectItem) Title() string {
	return i.obj.Name
}
func (i objectItem) Description() string {
	return fmt.Sprintf("%s  %s", i.obj.Type, i.obj.CreatedAt.Format("02.01.2006 15:04"))
}
func (i objectItem) FilterValue() string {
	return i.obj.Name
}

// Сообщения для асинхронных операций.
type objectsMsg struct {
	objects []*model.Object
	page    int
	pages   int
	err     error
}

type metadataMsg struct {
	metadata []*model.Metadata
	err      error
}

type statsMsg struct {
	stats *api.Stats
	err   error
}

type versionMsg struct {
	version string
	err     error
}

type plaintextMsg struct {
	plaintext string
	err       error
}

type keyMsg struct {
	key crypto.Key
	err error
}

type actionMsg struct {
	message string
	err     error
}

type tuiModel struct {
	app       *app
	clientVer string
	serverVer string

	state tuiState

	width  int
	height int

	// авторизация
	authLogin    textinput.Model
	authPassword textinput.Model
	authFocus    int

	// список
	objects []*model.Object
	list    list.Model
	page    int
	pages   int
	loading bool

	// выбранный объект
	selected  *model.Object
	metadata  []*model.Metadata
	reveal    bool
	plaintext string

	// главное меню
	menuIndex int

	// статистика
	stats *api.Stats

	// формы
	formType   model.SecretType
	formFields []string
	formValues []string
	formIndex  int
	formInput  textinput.Model
	editID     string

	// отложенная отправка формы (ожидание мастер-пароля)
	pendingName string
	pendingData any
	pendingPath string
	formSalt    []byte

	// запрос мастер-пароля
	passInput  textinput.Model
	passTarget string

	// скачивание
	dlInput textinput.Model
	dlObj   *model.Object

	err error
}

// RunTUI запускает TUI-интерфейс.
func RunTUI(a *app, clientVersion string) error {
	p := tea.NewProgram(newTUIModel(a, clientVersion), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func newTUIModel(a *app, clientVersion string) tuiModel {
	login := textinput.New()
	login.Placeholder = "логин"

	password := textinput.New()
	password.Placeholder = "пароль"
	password.EchoMode = textinput.EchoPassword

	pass := textinput.New()
	pass.Placeholder = "мастер-пароль"
	pass.EchoMode = textinput.EchoPassword

	dl := textinput.New()
	dl.Placeholder = "путь для сохранения"

	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 38, 20)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)

	state := stateAuth
	if _, err := a.requireToken(); err == nil {
		state = stateMain
	}

	return tuiModel{
		app:          a,
		clientVer:    clientVersion,
		state:        state,
		authLogin:    login,
		authPassword: password,
		authFocus:    0,
		list:         l,
		page:         1,
		pages:        1,
		passInput:    pass,
		dlInput:      dl,
		formInput:    textinput.New(),
	}
}

func (m tuiModel) Init() tea.Cmd {
	if m.state == stateMain {
		return tea.Batch(m.fetchObjects(1), m.fetchVersion())
	}
	m.authLogin.Focus()
	return nil
}

// ---- команды ----

func (m tuiModel) fetchObjects(page int) tea.Cmd {
	return func() tea.Msg {
		token, err := m.app.requireToken()
		if err != nil {
			return objectsMsg{err: err}
		}
		resp, err := m.app.api.ListObjectsPage(context.Background(), token, page, 10)
		if err != nil {
			return objectsMsg{err: err}
		}
		return objectsMsg{objects: resp.Data, page: page, pages: resp.Metadata.Pages}
	}
}

func (m tuiModel) fetchMetadata(objectID string) tea.Cmd {
	return func() tea.Msg {
		token, err := m.app.requireToken()
		if err != nil {
			return metadataMsg{err: err}
		}
		md, err := m.app.api.ListMetadata(context.Background(), token, objectID)
		if err != nil {
			return metadataMsg{err: err}
		}
		return metadataMsg{metadata: md}
	}
}

func (m tuiModel) fetchStats() tea.Cmd {
	return func() tea.Msg {
		token, err := m.app.requireToken()
		if err != nil {
			return statsMsg{err: err}
		}
		s, err := m.app.api.Stats(context.Background(), token)
		if err != nil {
			return statsMsg{err: err}
		}
		return statsMsg{stats: s}
	}
}

func (m tuiModel) fetchVersion() tea.Cmd {
	return func() tea.Msg {
		v, err := m.app.api.ServerVersion(context.Background())
		if err != nil {
			return versionMsg{err: err}
		}
		return versionMsg{version: v.Version}
	}
}

func (m tuiModel) doLogin(login, password string) tea.Cmd {
	return func() tea.Msg {
		resp, err := m.app.api.Login(context.Background(), login, password)
		if err != nil {
			return actionMsg{err: err}
		}
		if err := m.app.saveAuth(resp.Token, resp.Salt, password); err != nil {
			return actionMsg{err: err}
		}
		if err := m.app.sync.Pull(context.Background(), resp.Token); err != nil {
			return actionMsg{message: "вход выполнен, синхронизация не удалась: " + err.Error()}
		}
		return actionMsg{message: "ok"}
	}
}

func (m tuiModel) decryptObject(obj *model.Object, key crypto.Key) tea.Cmd {
	return func() tea.Msg {
		plain, err := crypto.Decrypt(key, obj.Ciphertext)
		if err != nil {
			return plaintextMsg{err: err}
		}
		text, err := formatDecrypted(obj.Type, plain, m.reveal)
		if err != nil {
			return plaintextMsg{err: err}
		}
		return plaintextMsg{plaintext: text}
	}
}

func (m tuiModel) doCreateObject(name string, typ model.SecretType, salt []byte, key crypto.Key, data any) tea.Cmd {
	return func() tea.Msg {
		token, err := m.app.requireToken()
		if err != nil {
			return actionMsg{err: err}
		}
		cipher, err := encryptPayloadKey(key, data)
		if err != nil {
			return actionMsg{err: err}
		}
		_, err = m.app.sync.CreateObject(context.Background(), token, api.CreateObjectRequest{
			Name: name, Type: typ, Salt: salt, Ciphertext: cipher,
		})
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{message: "объект создан"}
	}
}

func (m tuiModel) doUpdateObject(id, name string, typ model.SecretType, salt []byte, key crypto.Key, data any) tea.Cmd {
	return func() tea.Msg {
		token, err := m.app.requireToken()
		if err != nil {
			return actionMsg{err: err}
		}
		cipher, err := encryptPayloadKey(key, data)
		if err != nil {
			return actionMsg{err: err}
		}
		_, err = m.app.sync.UpdateObject(context.Background(), token, id, api.CreateObjectRequest{
			Name: name, Type: typ, Salt: salt, Ciphertext: cipher,
		})
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{message: "объект обновлён"}
	}
}

func (m tuiModel) doDeleteObject(id string) tea.Cmd {
	return func() tea.Msg {
		token, err := m.app.requireToken()
		if err != nil {
			return actionMsg{err: err}
		}
		if err := m.app.sync.DeleteObject(context.Background(), token, id); err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{message: "объект удалён"}
	}
}

func (m tuiModel) doBinaryUpload(name, path string, salt []byte, key crypto.Key) tea.Cmd {
	return func() tea.Msg {
		token, err := m.app.requireToken()
		if err != nil {
			return actionMsg{err: err}
		}
		if err := addBinaryFileKey(m.app, token, salt, key, name, path); err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{message: "файл загружен"}
	}
}

func (m tuiModel) doDownload(path string) tea.Cmd {
	return func() tea.Msg {
		if m.dlObj == nil || path == "" {
			return actionMsg{err: fmt.Errorf("укажите путь")}
		}
		token, err := m.app.requireToken()
		if err != nil {
			return actionMsg{err: err}
		}
		rc, _, err := m.app.api.DownloadFile(context.Background(), token, m.dlObj.ID)
		if err != nil {
			return actionMsg{err: err}
		}
		defer rc.Close()
		out, err := os.Create(path)
		if err != nil {
			return actionMsg{err: err}
		}
		defer out.Close()
		if _, err := io.Copy(out, rc); err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{message: "файл сохранён: " + path}
	}
}

func encryptPayloadKey(key crypto.Key, data any) ([]byte, error) {
	plaintext, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return crypto.Encrypt(key, plaintext)
}

// ---- Update ----

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.height > 5 {
			m.list.SetSize(38, m.height-5)
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case objectsMsg:
		if msg.err != nil {
			m.err = msg.err
			m.loading = false
			return m, nil
		}
		if msg.page == 1 {
			m.objects = msg.objects
		} else {
			m.objects = append(m.objects, msg.objects...)
		}
		m.page = msg.page
		m.pages = msg.pages
		m.loading = false
		m.rebuildList()
		return m, nil
	case metadataMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.metadata = msg.metadata
		return m, nil
	case statsMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.stats = msg.stats
		return m, nil
	case versionMsg:
		if msg.err == nil {
			m.serverVer = msg.version
		}
		return m, nil
	case plaintextMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.plaintext = msg.plaintext
		return m, nil
	case keyMsg:
		return m.handleKeyMsg(msg)
	case actionMsg:
		return m.handleActionMsg(msg)
	}
	return m, nil
}

func (m tuiModel) handleKeyMsg(msg keyMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.state = stateMain
		return m, nil
	}

	switch m.passTarget {
	case "form":
		if m.formType == model.SecretTypeBinary {
			return m, m.doBinaryUpload(m.pendingName, m.pendingPath, m.formSalt, msg.key)
		}
		if m.state == stateEdit && m.editID != "" {
			return m, m.doUpdateObject(m.editID, m.pendingName, m.formType, m.formSalt, msg.key, m.pendingData)
		}
		return m, m.doCreateObject(m.pendingName, m.formType, m.formSalt, msg.key, m.pendingData)
	default: // detail
		m.state = stateList
		if m.selected != nil {
			return m, tea.Batch(m.decryptObject(m.selected, msg.key), m.fetchMetadata(m.selected.ID))
		}
		return m, nil
	}
}

func (m tuiModel) handleActionMsg(msg actionMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}
	switch m.state {
	case stateAuth:
		m.err = nil
		m.state = stateMain
		return m, tea.Batch(m.fetchObjects(1), m.fetchVersion())
	case stateCreate, stateEdit, stateConfirmDelete:
		m.err = nil
		m.selected = nil
		m.state = stateList
		return m, m.fetchObjects(1)
	case stateDownload:
		m.err = nil
		m.state = stateList
		return m, nil
	}
	return m, nil
}

func (m *tuiModel) rebuildList() {
	items := make([]list.Item, 0, len(m.objects))
	for _, o := range m.objects {
		items = append(items, objectItem{obj: o})
	}
	m.list.SetItems(items)
}

func (m tuiModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.state {
	case stateAuth:
		return m.handleAuthKey(msg)
	case statePassword:
		return m.handlePasswordKey(msg)
	case stateMain:
		return m.handleMainKey(key)
	case stateList:
		return m.handleListKey(msg)
	case stateStats:
		return m.handleStatsKey(key)
	case stateCreate, stateEdit:
		return m.handleFormKey(msg)
	case stateConfirmDelete:
		return m.handleConfirmKey(key)
	case stateDownload:
		return m.handleDownloadKey(msg)
	}
	return m, nil
}

func (m tuiModel) handleAuthKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "tab":
		m.authFocus = (m.authFocus + 1) % 2
		m.focusAuthInputs()
		return m, nil
	case "enter":
		if m.authFocus == 0 {
			m.authFocus = 1
			m.focusAuthInputs()
			return m, nil
		}
		m.err = nil
		return m, m.doLogin(m.authLogin.Value(), m.authPassword.Value())
	}
	var cmd tea.Cmd
	if m.authFocus == 0 {
		m.authLogin, cmd = m.authLogin.Update(msg)
	} else {
		m.authPassword, cmd = m.authPassword.Update(msg)
	}
	return m, cmd
}

func (m *tuiModel) focusAuthInputs() {
	if m.authFocus == 0 {
		m.authLogin.Focus()
		m.authPassword.Blur()
	} else {
		m.authLogin.Blur()
		m.authPassword.Focus()
	}
}

func (m tuiModel) handlePasswordKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "enter":
		return m, m.submitMasterPassword(m.passInput.Value())
	case "esc":
		m.state = stateMain
		return m, nil
	}
	var cmd tea.Cmd
	m.passInput, cmd = m.passInput.Update(msg)
	return m, cmd
}

func (m tuiModel) submitMasterPassword(password string) tea.Cmd {
	return func() tea.Msg {
		salt, err := m.app.loadSalt()
		if err != nil {
			return keyMsg{err: err}
		}
		key, err := crypto.DeriveKey(password, salt)
		if err != nil {
			return keyMsg{err: err}
		}
		if err := m.app.saveMasterKey(key); err != nil {
			return keyMsg{err: err}
		}
		return keyMsg{key: key}
	}
}

func (m tuiModel) handleMainKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up":
		if m.menuIndex > 0 {
			m.menuIndex--
		}
		return m, nil
	case "down":
		if m.menuIndex < len(mainMenuItems)-1 {
			m.menuIndex++
		}
		return m, nil
	case "left":
		m.state = stateList
		if len(m.objects) == 0 {
			return m, m.fetchObjects(1)
		}
		return m, nil
	case "enter":
		return m.selectMenu()
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m tuiModel) selectMenu() (tea.Model, tea.Cmd) {
	switch mainMenuItems[m.menuIndex] {
	case "Создать":
		m.state = stateCreate
		m.err = nil
		m.startForm(model.SecretTypeText, false)
		return m, nil
	case "Статистика":
		m.state = stateStats
		m.err = nil
		return m, m.fetchStats()
	case "Выход":
		return m, tea.Quit
	}
	return m, nil
}

func (m tuiModel) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := strings.ToLower(msg.String())
	switch key {
	case "esc", "right":
		m.state = stateMain
		return m, nil
	case "r":
		if m.selected != nil && m.selected.Type != model.SecretTypeBinary {
			m.reveal = !m.reveal
			key, ok, err := m.app.cachedMasterKey()
			if err != nil {
				m.err = err
				return m, nil
			}
			if ok {
				return m, m.decryptObject(m.selected, key)
			}
			m.passTarget = "detail"
			m.state = statePassword
			m.passInput.Reset()
			m.passInput.Focus()
		}
		return m, nil
	case "e":
		if m.selected != nil && m.selected.Type != model.SecretTypeBinary {
			m.state = stateEdit
			m.editID = m.selected.ID
			m.err = nil
			m.startForm(m.selected.Type, true)
		}
		return m, nil
	case "delete":
		if m.selected != nil {
			m.state = stateConfirmDelete
			m.err = nil
		}
		return m, nil
	case "d":
		if m.selected != nil && m.selected.Type == model.SecretTypeBinary {
			m.state = stateDownload
			m.dlObj = m.selected
			m.dlInput.Reset()
			m.dlInput.Focus()
			m.err = nil
		}
		return m, nil
	default:
		m.list, _ = m.list.Update(msg)
		var cmd tea.Cmd
		m, cmd = m.selectItem()
		if cmd != nil {
			return m, cmd
		}
		if key == "down" && m.list.Index() >= len(m.objects)-1 && m.page < m.pages && !m.loading {
			m.loading = true
			return m, m.fetchObjects(m.page + 1)
		}
		return m, nil
	}
}

func (m tuiModel) selectItem() (tuiModel, tea.Cmd) {
	idx := m.list.Index()
	if idx < 0 || idx >= len(m.objects) {
		return m, nil
	}
	obj := m.objects[idx]
	if m.selected != nil && m.selected.ID == obj.ID {
		return m, nil
	}
	m.selected = obj
	m.plaintext = ""
	m.metadata = nil
	return m.loadSelected()
}

func (m tuiModel) loadSelected() (tuiModel, tea.Cmd) {
	obj := m.selected
	if obj.Type == model.SecretTypeBinary {
		return m, m.fetchMetadata(obj.ID)
	}
	key, ok, err := m.app.cachedMasterKey()
	if err != nil {
		m.err = err
		return m, nil
	}
	if ok {
		return m, tea.Batch(m.decryptObject(obj, key), m.fetchMetadata(obj.ID))
	}
	m.passTarget = "detail"
	m.state = statePassword
	m.passInput.Reset()
	m.passInput.Focus()
	return m, nil
}

func (m tuiModel) handleStatsKey(key string) (tea.Model, tea.Cmd) {
	key = strings.ToLower(key)
	switch key {
	case "h":
		m.state = stateMain
		return m, nil
	case "l":
		m.state = stateList
		m.list.Select(0)
		return m.selectItem()
	}
	return m, nil
}

func (m tuiModel) handleConfirmKey(key string) (tea.Model, tea.Cmd) {
	key = strings.ToLower(key)
	switch key {
	case "y", "enter":
		if m.selected != nil {
			return m, m.doDeleteObject(m.selected.ID)
		}
		m.state = stateList
		return m, nil
	case "n", "esc":
		m.state = stateList
		return m, nil
	}
	return m, nil
}

func (m tuiModel) handleDownloadKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "enter":
		return m, m.doDownload(m.dlInput.Value())
	case "esc":
		m.state = stateList
		return m, nil
	}
	var cmd tea.Cmd
	m.dlInput, cmd = m.dlInput.Update(msg)
	return m, cmd
}

// ---- формы ----

func (m *tuiModel) startForm(typ model.SecretType, editing bool) {
	m.formType = typ
	m.formIndex = 0
	m.formInput = textinput.New()
	m.formInput.Focus()

	switch typ {
	case model.SecretTypeText:
		m.formFields = []string{"имя", "текст"}
	case model.SecretTypeLoginPassword:
		m.formFields = []string{"имя", "логин", "пароль"}
	case model.SecretTypeCard:
		m.formFields = []string{"имя", "номер", "держатель", "месяц (MM)", "год (YYYY)", "cvv"}
	case model.SecretTypeBinary:
		m.formFields = []string{"имя", "путь к файлу"}
	default:
		m.formFields = []string{"имя"}
	}

	m.formValues = nil
	if editing && m.selected != nil {
		m.formValues = []string{m.selected.Name}
		m.formIndex = 1
	}

	m.formInput.Placeholder = m.formFields[m.formIndex]
	if m.formIndex < len(m.formValues) {
		m.formInput.SetValue(m.formValues[m.formIndex])
	}
}

func (m tuiModel) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		m.state = stateMain
		return m, nil
	case "enter":
		m.formValues = append(m.formValues, m.formInput.Value())
		m.formIndex++
		if m.formIndex < len(m.formFields) {
			m.formInput.Reset()
			m.formInput.Placeholder = m.formFields[m.formIndex]
			return m, nil
		}
		return m.submitForm()
	}
	var cmd tea.Cmd
	m.formInput, cmd = m.formInput.Update(msg)
	return m, cmd
}

func (m tuiModel) submitForm() (tea.Model, tea.Cmd) {
	salt, err := m.app.loadSalt()
	if err != nil {
		m.err = err
		return m, nil
	}
	m.formSalt = salt

	name := m.formValues[0]
	switch m.formType {
	case model.SecretTypeBinary:
		m.pendingName = name
		m.pendingPath = m.formValues[1]
		return m.requireKeyForForm()
	case model.SecretTypeText:
		m.pendingData = model.TextData{Content: m.formValues[1]}
	case model.SecretTypeLoginPassword:
		m.pendingData = model.LoginPasswordData{Login: m.formValues[1], Password: m.formValues[2]}
	case model.SecretTypeCard:
		month, _ := strconv.Atoi(m.formValues[3])
		year, _ := strconv.Atoi(m.formValues[4])
		m.pendingData = model.CardData{Number: m.formValues[1], Holder: m.formValues[2], ExpMonth: month, ExpYear: year, CVV: m.formValues[5]}
	}
	m.pendingName = name
	return m.requireKeyForForm()
}

func (m tuiModel) requireKeyForForm() (tea.Model, tea.Cmd) {
	key, ok, err := m.app.cachedMasterKey()
	if err != nil {
		m.err = err
		return m, nil
	}
	if ok {
		return m.submitPending(key)
	}
	m.passTarget = "form"
	m.state = statePassword
	m.passInput.Reset()
	m.passInput.Focus()
	return m, nil
}

func (m tuiModel) submitPending(key crypto.Key) (tea.Model, tea.Cmd) {
	if m.formType == model.SecretTypeBinary {
		return m, m.doBinaryUpload(m.pendingName, m.pendingPath, m.formSalt, key)
	}
	if m.state == stateEdit && m.editID != "" {
		return m, m.doUpdateObject(m.editID, m.pendingName, m.formType, m.formSalt, key, m.pendingData)
	}
	return m, m.doCreateObject(m.pendingName, m.formType, m.formSalt, key, m.pendingData)
}

// ---- View ----

func (m tuiModel) View() string {
	switch m.state {
	case stateAuth:
		return m.viewAuth()
	case statePassword:
		return m.viewPassword()
	case stateDownload:
		return m.viewDownload()
	case stateCreate, stateEdit:
		return m.viewForm()
	}

	left := lipgloss.NewStyle().Width(40).Border(lipgloss.RoundedBorder()).Render(m.list.View())
	right := lipgloss.NewStyle().Width(70).Border(lipgloss.RoundedBorder()).Padding(0, 1).Render(m.viewRight())
	hint := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1).
		Render(m.viewHint())

	content := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return lipgloss.JoinVertical(lipgloss.Left, content, hint)
}

func (m tuiModel) viewAuth() string {
	var b strings.Builder
	fmt.Fprintf(&b, "GophKeeper TUI — вход\n")
	fmt.Fprintf(&b, "версия клиента: %s\n\n", m.clientVer)
	fmt.Fprintf(&b, "Логин:\n%s\n\n", m.authLogin.View())
	fmt.Fprintf(&b, "Пароль:\n%s\n\n", m.authPassword.View())
	if m.err != nil {
		fmt.Fprintf(&b, "ошибка: %s\n", m.err.Error())
	}
	fmt.Fprintf(&b, "\n[tab] переключить  [enter] войти  [ctrl+c] выход\n")
	return b.String()
}

func (m tuiModel) viewPassword() string {
	var b strings.Builder
	b.WriteString("Введите мастер-пароль\n\n")
	b.WriteString(m.passInput.View() + "\n\n")
	if m.err != nil {
		fmt.Fprintf(&b, "ошибка: %s\n", m.err.Error())
	}
	b.WriteString("\n[enter] подтвердить  [esc] отмена\n")
	return b.String()
}

func (m tuiModel) viewDownload() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Скачать файл: %s\n\n", m.dlObj.Name)
	b.WriteString(m.dlInput.View() + "\n\n")
	b.WriteString("[enter] скачать  [esc] отмена\n")
	return b.String()
}

func (m tuiModel) viewForm() string {
	var b strings.Builder
	if m.state == stateEdit {
		b.WriteString("Редактирование объекта\n\n")
	} else {
		b.WriteString("Создание объекта\n\n")
	}
	fmt.Fprintf(&b, "Поле (%d/%d): %s\n\n", m.formIndex+1, len(m.formFields), m.formFields[m.formIndex])
	b.WriteString(m.formInput.View() + "\n\n")
	if m.err != nil {
		fmt.Fprintf(&b, "ошибка: %s\n", m.err.Error())
	}
	b.WriteString("\n[enter] далее  [esc] отмена\n")
	return b.String()
}

func (m tuiModel) viewRight() string {
	var content string
	switch m.state {
	case stateMain:
		content = m.viewMainMenu()
	case stateStats:
		content = m.viewStats()
	default:
		content = m.viewObjectDetail()
	}
	if m.serverVer != "" {
		content = "версия сервера: " + m.serverVer + "\n\n" + content
	}
	return content
}

func (m tuiModel) viewMainMenu() string {
	var b strings.Builder
	b.WriteString("Главное меню\n\n")
	for i, item := range mainMenuItems {
		cursor := "  "
		if i == m.menuIndex {
			cursor = "> "
		}
		fmt.Fprintf(&b, "%s%s\n", cursor, item)
	}
	return b.String()
}

func (m tuiModel) viewStats() string {
	var b strings.Builder
	b.WriteString("Статистика\n\n")
	if m.stats == nil {
		b.WriteString("загрузка...\n")
	} else {
		fmt.Fprintf(&b, "объектов в БД: %d\n", m.stats.ObjectsCount)
		fmt.Fprintf(&b, "файлов в хранилище: %d\n", m.stats.FilesCount)
		fmt.Fprintf(&b, "общий размер файлов: %d байт\n", m.stats.FilesTotalSize)
	}
	return b.String()
}

func (m tuiModel) viewObjectDetail() string {
	if m.selected == nil {
		return "выберите объект\n"
	}
	obj := m.selected
	var b strings.Builder
	fmt.Fprintf(&b, "id: %s\n", obj.ID)
	fmt.Fprintf(&b, "имя: %s\n", obj.Name)
	fmt.Fprintf(&b, "тип: %s\n", obj.Type)
	fmt.Fprintf(&b, "создан: %s\n", obj.CreatedAt.Format("02.01.2006 15:04"))
	fmt.Fprintf(&b, "обновлён: %s\n", obj.UpdatedAt.Format("02.01.2006 15:04"))
	b.WriteString("\n")

	if obj.Type == model.SecretTypeBinary {
		var meta binaryMeta
		if err := json.Unmarshal(obj.Ciphertext, &meta); err == nil {
			b.WriteString(formatBinaryMeta(meta) + "\n")
		}
	} else if m.plaintext != "" {
		b.WriteString(m.plaintext + "\n")
	}

	b.WriteString("\nМетаданные:\n" + formatMetadata(m.metadata) + "\n")
	return b.String()
}

func (m tuiModel) viewHint() string {
	switch m.state {
	case stateMain:
		return "↑/↓ — навигация   Enter — выбрать   ← — к списку   Q — выход"
	case stateStats:
		return "H — на главную   L — к списку объектов"
	case stateList:
		hint := "↑/↓ — список   ←/→ — фокус   R — показать скрытое   E — редактировать   Delete — удалить"
		if m.selected != nil && m.selected.Type == model.SecretTypeBinary {
			hint += "   D — скачать файл"
		}
		return hint
	default:
		return ""
	}
}
