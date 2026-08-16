package main

import (
	"fmt"

	"github.com/victor/gophkeeper/internal/client/api"
)

// cmdAdd обрабатывает команду add (добавление объекта).
func cmdAdd(serverURL string) {
	a := mustApp(serverURL)
	defer a.close()

	token, err := a.requireToken()
	if err != nil {
		fatal("%v", err)
	}
	salt, err := a.loadSalt()
	if err != nil {
		fatal("%v", err)
	}

	typ := promptType()

	name, err := prompt("имя: ")
	if err != nil {
		fatal("не удалось прочитать имя: %v", err)
	}
	if name == "" {
		fatal("имя обязательно")
	}

	data, err := readPayloadForType(typ)
	if err != nil {
		fatal("%v", err)
	}

	master, err := promptSecret("мастер-пароль: ")
	if err != nil {
		fatal("не удалось прочитать мастер-пароль: %v", err)
	}

	ciphertext, err := encryptPayload(master, salt, data)
	if err != nil {
		fatal("не удалось зашифровать данные: %v", err)
	}

	obj, err := a.sync.CreateObject(ctx(), token, api.CreateObjectRequest{
		Name:       name,
		Type:       typ,
		Salt:       salt,
		Ciphertext: ciphertext,
	})
	if err != nil {
		fatal("не удалось создать объект: %v", err)
	}
	fmt.Printf("создан объект %s\n", obj.ID)
}

// cmdList обрабатывает команду list (список объектов из кэша).
func cmdList(serverURL string) {
	a := mustApp(serverURL)
	defer a.close()

	objects, err := a.store.ListObjects(ctx())
	if err != nil {
		fatal("не удалось прочитать кэш: %v", err)
	}

	if len(objects) == 0 {
		fmt.Println("кэш пуст — выполните login или sync")
		return
	}

	for _, o := range objects {
		fmt.Printf("%s  %-20s  %s\n", o.ID, o.Name, o.Type)
	}
}

// cmdGet обрабатывает команду get (показ объекта).
func cmdGet(serverURL string, args []string) {
	if len(args) == 0 {
		fatal("укажите идентификатор объекта: get <id>")
	}
	id := args[0]

	a := mustApp(serverURL)
	defer a.close()

	token, err := a.requireToken()
	if err != nil {
		fatal("%v", err)
	}
	salt, err := a.loadSalt()
	if err != nil {
		fatal("%v", err)
	}

	obj, err := a.sync.GetObject(ctx(), token, id)
	if err != nil {
		fatal("не удалось получить объект: %v", err)
	}

	master, err := promptSecret("мастер-пароль: ")
	if err != nil {
		fatal("не удалось прочитать мастер-пароль: %v", err)
	}

	plaintext, err := decryptPayload(master, salt, obj.Ciphertext)
	if err != nil {
		fatal("не удалось расшифровать данные: %v", err)
	}

	out, err := formatPayload(obj.Type, plaintext)
	if err != nil {
		fatal("не удалось разобрать данные: %v", err)
	}

	fmt.Printf("имя: %s\n", obj.Name)
	fmt.Println(out)
}

// cmdEdit обрабатывает команду edit (изменение объекта).
func cmdEdit(serverURL string, args []string) {
	if len(args) == 0 {
		fatal("укажите идентификатор объекта: edit <id>")
	}
	id := args[0]

	a := mustApp(serverURL)
	defer a.close()

	token, err := a.requireToken()
	if err != nil {
		fatal("%v", err)
	}
	salt, err := a.loadSalt()
	if err != nil {
		fatal("%v", err)
	}

	obj, err := a.sync.GetObject(ctx(), token, id)
	if err != nil {
		fatal("не удалось получить объект: %v", err)
	}

	name, err := prompt(fmt.Sprintf("имя [%s]: ", obj.Name))
	if err != nil {
		fatal("не удалось прочитать имя: %v", err)
	}
	if name == "" {
		name = obj.Name
	}

	data, err := readPayloadForType(obj.Type)
	if err != nil {
		fatal("%v", err)
	}

	master, err := promptSecret("мастер-пароль: ")
	if err != nil {
		fatal("не удалось прочитать мастер-пароль: %v", err)
	}

	ciphertext, err := encryptPayload(master, salt, data)
	if err != nil {
		fatal("не удалось зашифровать данные: %v", err)
	}

	updated, err := a.sync.UpdateObject(ctx(), token, id, api.CreateObjectRequest{
		Name:       name,
		Type:       obj.Type,
		Salt:       salt,
		Ciphertext: ciphertext,
	})
	if err != nil {
		fatal("не удалось обновить объект: %v", err)
	}
	fmt.Printf("объект %s обновлён\n", updated.ID)
}

// cmdDelete обрабатывает команду delete (удаление объекта).
func cmdDelete(serverURL string, args []string) {
	if len(args) == 0 {
		fatal("укажите идентификатор объекта: delete <id>")
	}
	id := args[0]

	a := mustApp(serverURL)
	defer a.close()

	token, err := a.requireToken()
	if err != nil {
		fatal("%v", err)
	}

	if err := a.sync.DeleteObject(ctx(), token, id); err != nil {
		fatal("не удалось удалить объект: %v", err)
	}
	fmt.Printf("объект %s удалён\n", id)
}

// cmdSync обрабатывает команду sync (полная синхронизация с сервером).
func cmdSync(serverURL string) {
	a := mustApp(serverURL)
	defer a.close()

	token, err := a.requireToken()
	if err != nil {
		fatal("%v", err)
	}

	if err := a.sync.Pull(ctx(), token); err != nil {
		fatal("синхронизация не удалась: %v", err)
	}
	fmt.Println("данные синхронизированы")
}
