package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/victor/gophkeeper/internal/model"
)

// Замаскированные значения для отображения скрытых элементов.
const (
	maskedPassword = "*****"
	maskedCVV      = "***"
	maskedExpiry   = "**/**"
)

// last4 возвращает последние 4 символа строки (или строку целиком, если она короче).
func last4(s string) string {
	if len(s) <= 4 {
		return s
	}
	return s[len(s)-4:]
}

// maskCardNumber показывает только последние 4 цифры номера карты.
func maskCardNumber(number string) string {
	if len(number) <= 4 {
		return number
	}
	return "..." + last4(number)
}

// formatDecrypted преобразует расшифрованные данные объекта в строку для вывода.
func formatDecrypted(typ model.SecretType, plaintext []byte, reveal bool) (string, error) {
	switch typ {
	case model.SecretTypeLoginPassword:
		var d model.LoginPasswordData
		if err := json.Unmarshal(plaintext, &d); err != nil {
			return "", err
		}
		pw := maskedPassword
		if reveal {
			pw = d.Password
		}
		return fmt.Sprintf("логин: %s\nпароль: %s", d.Login, pw), nil
	case model.SecretTypeText:
		var d model.TextData
		if err := json.Unmarshal(plaintext, &d); err != nil {
			return "", err
		}
		return d.Content, nil
	case model.SecretTypeCard:
		var d model.CardData
		if err := json.Unmarshal(plaintext, &d); err != nil {
			return "", err
		}
		number := maskCardNumber(d.Number)
		cvv := maskedCVV
		exp := maskedExpiry
		if reveal {
			number = d.Number
			cvv = d.CVV
			exp = fmt.Sprintf("%02d/%d", d.ExpMonth, d.ExpYear)
		}
		return fmt.Sprintf("номер: %s\nдержатель: %s\nсрок: %s\ncvv: %s", number, d.Holder, exp, cvv), nil
	default:
		return string(plaintext), nil
	}
}

// formatMetadata преобразует список метаданных в строки для вывода.
func formatMetadata(metadata []*model.Metadata) string {
	if len(metadata) == 0 {
		return "(метаданных нет)"
	}
	var b strings.Builder
	for _, m := range metadata {
		fmt.Fprintf(&b, "- %s", m.Name)
		if len(m.Options) > 0 {
			fmt.Fprintf(&b, ": %v", m.Options)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// formatBinaryMeta преобразует метаданные бинарного файла в строки.
func formatBinaryMeta(meta binaryMeta) string {
	return fmt.Sprintf("размер: %d байт\nmimetype: %s\nsha256: %s", meta.Size, meta.ContentType, meta.SHA256)
}
