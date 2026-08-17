package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/victor/gophkeeper/internal/crypto"
	"github.com/victor/gophkeeper/internal/model"
)

// encryptPayload шифрует открытый текст производным ключом.
func encryptPayload(key crypto.Key, data any) ([]byte, error) {
	plaintext, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return crypto.Encrypt(key, plaintext)
}

// decryptPayload расшифровывает шифротекст производным ключом.
func decryptPayload(key crypto.Key, ciphertext []byte) ([]byte, error) {
	return crypto.Decrypt(key, ciphertext)
}

// formatPayload преобразует расшифрованный открытый текст в читаемый вид.
func formatPayload(typ model.SecretType, plaintext []byte) (string, error) {
	switch typ {
	case model.SecretTypeLoginPassword:
		var d model.LoginPasswordData
		if err := json.Unmarshal(plaintext, &d); err != nil {
			return "", err
		}
		return fmt.Sprintf("login: %s\npassword: %s", d.Login, d.Password), nil
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
		return fmt.Sprintf("number: %s\nholder: %s\nexp: %02d/%d\ncvv: %s",
			d.Number, d.Holder, d.ExpMonth, d.ExpYear, d.CVV), nil
	default:
		return string(plaintext), nil
	}
}

// readPayloadForType запрашивает у пользователя поля в зависимости от типа.
func readPayloadForType(typ model.SecretType) (any, error) {
	switch typ {
	case model.SecretTypeLoginPassword:
		login, err := prompt("логин: ")
		if err != nil {
			return nil, err
		}
		password, err := promptSecret("пароль: ")
		if err != nil {
			return nil, err
		}
		return model.LoginPasswordData{Login: login, Password: password}, nil
	case model.SecretTypeText:
		content, err := prompt("текст: ")
		if err != nil {
			return nil, err
		}
		return model.TextData{Content: content}, nil
	case model.SecretTypeCard:
		number, err := prompt("номер карты: ")
		if err != nil {
			return nil, err
		}
		holder, err := prompt("держатель: ")
		if err != nil {
			return nil, err
		}
		expMonthStr, err := prompt("месяц (MM): ")
		if err != nil {
			return nil, err
		}
		expYearStr, err := prompt("год (YYYY): ")
		if err != nil {
			return nil, err
		}
		cvv, err := promptSecret("cvv: ")
		if err != nil {
			return nil, err
		}
		expMonth, err := strconv.Atoi(expMonthStr)
		if err != nil {
			return nil, fmt.Errorf("неверный месяц: %s", expMonthStr)
		}
		expYear, err := strconv.Atoi(expYearStr)
		if err != nil {
			return nil, fmt.Errorf("неверный год: %s", expYearStr)
		}
		return model.CardData{Number: number, Holder: holder, ExpMonth: expMonth, ExpYear: expYear, CVV: cvv}, nil
	default:
		return nil, fmt.Errorf("неизвестный тип объекта: %s", typ)
	}
}

// promptType запрашивает тип объекта до тех пор, пока не введён корректный.
func promptType() model.SecretType {
	for {
		s, err := prompt("тип (login_password|text|card|binary): ")
		if err != nil {
			fatal("не удалось прочитать тип: %v", err)
		}
		typ := model.SecretType(s)
		if typ.Valid() {
			return typ
		}
		fmt.Println("неверный тип, попробуйте ещё раз")
	}
}
