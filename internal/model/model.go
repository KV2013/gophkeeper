// Package model содержит доменные типы системы GophKeeper.
//
// Данные пользователя хранятся на сервере только в зашифрованном виде:
// шифрование выполняется на клиенте (оконечное шифрование), поэтому все
// секретные поля объектов представлены непрозрачным шифротекстом Ciphertext.
package model

import "time"

// SecretType описывает тип хранимого объекта.
type SecretType string

// Поддерживаемые типы объектов.
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

// Valid возвращает true, если тип входит в список поддерживаемых.
func (t SecretType) Valid() bool {
	switch t {
	case SecretTypeLoginPassword, SecretTypeText, SecretTypeBinary, SecretTypeCard:
		return true
	default:
		return false
	}
}

// User — пользователь системы.
type User struct {
	// ID — уникальный идентификатор пользователя.
	ID string `json:"id" db:"id"`
	// Login — уникальный логин.
	Login string `json:"login" db:"login"`
	// PasswordHash — bcrypt-хеш пароля (не сериализуется в JSON).
	PasswordHash string `json:"-" db:"password_hash"`
	// Salt — соль KDF, используемая клиентом для вывода мастер-ключа.
	Salt []byte `json:"salt" db:"salt"`
	// CreatedAt — время регистрации.
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Object — зашифрованный объект данных пользователя.
//
// Поля Type не шифруется и доступно серверу. Поле Ciphertext содержит
// nonce || ciphertext, зашифрованный мастер-ключом клиента, и не может быть
// прочитано сервером. Поле Salt хранит соль KDF, необходимую клиенту для
// повторного вывода мастер-ключа.
type Object struct {
	// ID — уникальный идентификатор объекта.
	ID string `json:"id" db:"id"`
	// UserID — идентификатор владельца объекта.
	UserID string `json:"user_id" db:"user_id"`
	// Name — человекочитаемое имя объекта (открытый текст для списка/поиска).
	Name string `json:"name" db:"name"`
	// Type — тип хранимого объекта.
	Type SecretType `json:"type" db:"type"`
	// Salt — соль KDF (размер crypto.SaltSize), общая для пользователя.
	Salt []byte `json:"salt" db:"salt"`
	// Ciphertext — зашифрованные данные: nonce || ciphertext (base64 в JSON).
	Ciphertext []byte `json:"ciphertext" db:"ciphertext"`
	// CreatedAt — время создания объекта.
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	// UpdatedAt — время последнего изменения объекта.
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Metadata — метаданные объекта.
//
// Метаданные не шифруются и доступны серверу: по ним выполняется поиск и
// отображение списка объектов без доступа к их содержимому.
type Metadata struct {
	// ID — уникальный идентификатор записи метаданных.
	ID string `json:"id" db:"id"`
	// UserID — идентификатор владельца.
	UserID string `json:"user_id" db:"user_id"`
	// ObjectID — идентификатор объекта, к которому относятся метаданные.
	ObjectID string `json:"object_id" db:"object_id"`
	// Name — имя метаданных.
	Name string `json:"name" db:"name"`
	// OrderNumber — порядковый номер для сортировки.
	OrderNumber int `json:"order_number" db:"order_number"`
	// Options — произвольные пары ключ/значение.
	Options map[string]string `json:"options"`
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
