// Package model содержит доменные типы системы GophKeeper.
//
// Данные пользователя хранятся на сервере только в зашифрованном виде:
// шифрование выполняется на клиенте (оконечное шифрование), поэтому все
// секретные поля объектов представлены непрозрачным шифротекстом Ciphertext.
package model

import "time"

// SecretType описывает тип хранимого секрета.
type SecretType string

// Поддерживаемые типы секретов.
const (
	// SecretTypeLoginPassword — пара логин/пароль.
	SecretTypeLoginPassword SecretType = "login_password"
	// SecretTypeText — произвольные текстовые данные.
	SecretTypeText SecretType = "text"
	// SecretTypeBinary — произвольные бинарные данные.
	SecretTypeBinary SecretType = "binary"
	// SecretTypeCard — данные банковской карты.
	SecretTypeCard SecretType = "card"
)

// Metadata описывает незашифрованные метаданные объекта, по которым сервер
// выполняет поиск и отображение списка объектов без доступа к содержимому.
type Metadata struct {
	// Name — человекочитаемое название объекта.
	Name string `json:"name"`
	// Description — произвольное описание.
	Description string `json:"description,omitempty"`
	// Tags — произвольные теги для организации и поиска.
	Tags []string `json:"tags,omitempty"`
}

// Secret — универсальная модель хранимого секрета.
//
// Поля Type и Metadata не шифруются и доступны серверу. Поле Ciphertext
// содержит nonce || ciphertext, зашифрованный мастер-ключом клиента, и не
// может быть прочитан сервером. Поле Salt хранит соль KDF пользователя,
// необходимую клиенту для повторного вывода мастер-ключа.
type Secret struct {
	// ID — уникальный идентификатор объекта.
	ID string `json:"id"`
	// UserID — идентификатор владельца объекта.
	UserID string `json:"user_id"`
	// Type — тип хранимого секрета.
	Type SecretType `json:"type"`
	// Salt — соль KDF (размер crypto.SaltSize), общая для пользователя.
	Salt []byte `json:"salt"`
	// Ciphertext — зашифрованные данные: nonce || ciphertext (base64 в JSON).
	Ciphertext []byte `json:"ciphertext"`
	// Metadata — незашифрованные метаданные объекта.
	Metadata Metadata `json:"metadata"`
	// CreatedAt — время создания объекта.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt — время последнего изменения объекта.
	UpdatedAt time.Time `json:"updated_at"`
}

// LoginPasswordData — открытый текст пары логин/пароль на стороне клиента.
type LoginPasswordData struct {
	// Login — логин.
	Login string `json:"login"`
	// Password — пароль.
	Password string `json:"password"`
}

// TextData — открытый текст произвольных текстовых данных на стороне клиента.
type TextData struct {
	// Content — содержимое.
	Content string `json:"content"`
}

// BinaryData — открытый текст бинарных данных на стороне клиента.
type BinaryData struct {
	// Filename — имя файла.
	Filename string `json:"filename"`
	// ContentType — MIME-тип содержимого.
	ContentType string `json:"content_type"`
	// Content — содержимое файла.
	Content []byte `json:"content"`
}

// CardData — открытый текст данных банковской карты на стороне клиента.
type CardData struct {
	// Number — номер карты.
	Number string `json:"number"`
	// Holder — имя держателя.
	Holder string `json:"holder"`
	// ExpMonth — месяц окончания срока действия.
	ExpMonth int `json:"exp_month"`
	// ExpYear — год окончания срока действия.
	ExpYear int `json:"exp_year"`
	// CVV — код CVV/CVC.
	CVV string `json:"cvv"`
}
