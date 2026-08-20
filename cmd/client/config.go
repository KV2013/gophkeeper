package main

import (
	"errors"
	"fmt"
	"strconv"

	clientconfig "github.com/victor/gophkeeper/internal/client/config"
	"github.com/victor/gophkeeper/internal/client/repository"
)

// cmdConfigSet обрабатывает команду config-set.
func cmdConfigSet(serverURL string, args []string) {
	if len(args) < 2 {
		fatal("использование: config-set <key> <value>")
	}
	key, value := args[0], args[1]

	a := mustApp(serverURL)
	defer a.close()

	switch key {
	case clientconfig.KeyMasterKeyTTL:
		if _, err := clientconfig.ParseTTL(value); err != nil {
			fatal("%v", err)
		}
	case clientconfig.KeyUseCredentialsFile:
		if _, err := strconv.ParseBool(value); err != nil {
			fatal("неверное значение %s: %v", key, err)
		}
	}

	if err := a.store.SetConfig(ctx(), key, value); err != nil {
		fatal("не удалось сохранить конфиг: %v", err)
	}
	fmt.Printf("%s = %s\n", key, value)
}

// cmdConfigGet обрабатывает команду config-get.
func cmdConfigGet(serverURL string, args []string) {
	if len(args) == 0 {
		fatal("использование: config-get <key>")
	}
	key := args[0]

	a := mustApp(serverURL)
	defer a.close()

	v, err := a.store.GetConfig(ctx(), key)
	if errors.Is(err, repository.ErrNotFound) {
		fmt.Printf("%s: не задан\n", key)
		return
	}
	if err != nil {
		fatal("не удалось прочитать конфиг: %v", err)
	}
	fmt.Printf("%s = %s\n", key, v)
}
