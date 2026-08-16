// GophKeeper — клиентское CLI-приложение менеджера паролей.
package main

import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"
)

var (
	buildVersion = "dev"
	buildDate    = "unknown"
	buildCommit  = "unknown"
)

// Параметры TLS, задаваемые через флаги и используемые при создании клиента.
var (
	caCertPath string
	insecure   bool
)

func main() {
	root := flag.NewFlagSet("gophkeeper", flag.ExitOnError)
	root.Usage = usage
	server := root.String("server", defaultServer(), "адрес сервера (например http://localhost:8080)")
	root.StringVar(&caCertPath, "cacert", "", "путь к CA-сертификату сервера (PEM)")
	root.BoolVar(&insecure, "insecure", false, "не проверять TLS-сертификат (только для разработки)")
	_ = root.Parse(os.Args[1:])

	args := root.Args()
	if len(args) == 0 {
		root.Usage()
		os.Exit(2)
	}

	switch args[0] {
	case "version":
		printBuildInfo()
	case "register":
		cmdRegister(*server, args[1:])
	case "login":
		cmdLogin(*server, args[1:])
	case "logout":
		cmdLogout(*server)
	case "daemon":
		cmdDaemon(*server)
	case "config-set":
		cmdConfigSet(*server, args[1:])
	case "config-get":
		cmdConfigGet(*server, args[1:])
	case "add":
		cmdAdd(*server)
	case "list":
		cmdList(*server)
	case "get":
		cmdGet(*server, args[1:])
	case "edit":
		cmdEdit(*server, args[1:])
	case "delete":
		cmdDelete(*server, args[1:])
	case "sync":
		cmdSync(*server)
	default:
		fmt.Fprintf(os.Stderr, "неизвестная команда: %s\n\n", args[0])
		root.Usage()
		os.Exit(2)
	}
}

// usage выводит справку по командам.
func usage() {
	fmt.Fprint(os.Stderr, `GophKeeper — менеджер паролей (CLI).

Использование:
  gophkeeper [--server URL] [--cacert <путь>] [--insecure] <команда> [аргументы]

Флаги:
  --server URL   адрес сервера (например https://localhost:8080)
  --cacert путь  доверять CA-сертификату сервера из PEM-файла
  --insecure     не проверять TLS-сертификат (только для разработки)

Команды:
  register    зарегистрировать пользователя
  login       войти и синхронизировать данные
  logout      завершить сессию (удалить токен)
  daemon      запустить фоновый процесс очистки протухших секретов
  config-set <key> <value>  задать параметр конфигурации
  config-get <key>          показать параметр конфигурации
  add         добавить объект
  list        показать список объектов (из локального кэша)
  get <id>    показать объект
  edit <id>   изменить объект
  delete <id> удалить объект
  sync        синхронизировать данные с сервером
  version     показать версию и дату сборки
`)
}

// defaultServer возвращает адрес сервера из env или значение по умолчанию.
func defaultServer() string {
	if v := os.Getenv("GOPHKEEPER_SERVER"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

// fatal печатает ошибку в stderr и завершает процесс с кодом 1.
func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ошибка: "+format+"\n", args...)
	os.Exit(1)
}

// printBuildInfo выводит информацию о сборке.
func printBuildInfo() {
	fmt.Printf("Build version: %s\n", nA(buildVersion))
	fmt.Printf("Build date: %s\n", nA(buildDate))
	fmt.Printf("Build commit: %s\n", nA(buildCommit))
}

// nA заменяет пустую строку на "N/A".
func nA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}
