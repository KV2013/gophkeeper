package main

import (
	"fmt"

	flag "github.com/spf13/pflag"
)

// cmdRegister обрабатывает команду register.
func cmdRegister(serverURL string, args []string) {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	login := fs.StringP("login", "u", "", "логин")
	_ = fs.Parse(args)

	l := *login
	if l == "" {
		var err error
		if l, err = prompt("логин: "); err != nil {
			fatal("не удалось прочитать логин: %v", err)
		}
	}
	if l == "" {
		fatal("логин не может быть пустым")
	}

	password, err := promptSecret("пароль: ")
	if err != nil {
		fatal("не удалось прочитать пароль: %v", err)
	}
	if password == "" {
		fatal("пароль не может быть пустым")
	}

	a := mustApp(serverURL)
	defer a.close()

	resp, err := a.api.Register(ctx(), l, password)
	if err != nil {
		fatal("регистрация не удалась: %v", err)
	}
	if err := a.saveAuth(resp.Token, resp.Salt); err != nil {
		fatal("не удалось сохранить токен: %v", err)
	}

	fmt.Printf("пользователь %q зарегистрирован\n", l)
}

// cmdLogin обрабатывает команду login.
func cmdLogin(serverURL string, args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	login := fs.StringP("login", "u", "", "логин")
	_ = fs.Parse(args)

	l := *login
	if l == "" {
		var err error
		if l, err = prompt("логин: "); err != nil {
			fatal("не удалось прочитать логин: %v", err)
		}
	}
	if l == "" {
		fatal("логин не может быть пустым")
	}

	password, err := promptSecret("пароль: ")
	if err != nil {
		fatal("не удалось прочитать пароль: %v", err)
	}

	a := mustApp(serverURL)
	defer a.close()

	resp, err := a.api.Login(ctx(), l, password)
	if err != nil {
		fatal("вход не удался: %v", err)
	}
	if err := a.saveAuth(resp.Token, resp.Salt); err != nil {
		fatal("не удалось сохранить токен: %v", err)
	}

	if err := a.sync.Pull(ctx(), resp.Token); err != nil {
		fmt.Printf("вход выполнен, но синхронизация не удалась: %v\n", err)
	} else {
		fmt.Println("вход выполнен, данные синхронизированы")
	}
}
