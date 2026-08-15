// Package errors содержит общие ошибки хранилища данных.
package errors

import goerrors "errors"

// Ошибки, возвращаемые реализациями хранилища.
var (
	// ErrNotFound — запрошенный объект не найден.
	ErrNotFound = goerrors.New("repository: объект не найден")
	// ErrLoginExists — логин уже занят другим пользователем.
	ErrLoginExists = goerrors.New("repository: логин уже занят")
)
