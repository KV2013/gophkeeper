// Package service содержит бизнес-логику сервера: аутентификацию и
// управление объектами и их метаданными.
package service

import "errors"

// Ошибки, возвращаемые сервисами.
var (
	// ErrInvalidCredentials — неверный логин или пароль.
	ErrInvalidCredentials = errors.New("service: неверный логин или пароль")
	// ErrBadRequest — некорректные входные данные.
	ErrBadRequest = errors.New("service: некорректные входные данные")
	// ErrNotFound — объект не найден.
	ErrNotFound = errors.New("service: объект не найден")
)
