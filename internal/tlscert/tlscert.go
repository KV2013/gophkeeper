// Package tlscert предоставляет самоподписные TLS-сертификаты для dev-окружения.
package tlscert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// ErrNotGenerated возвращается при невозможности создать сертификат.
var ErrNotGenerated = errors.New("tlscert: не удалось сгенерировать сертификат")

// Имена файлов сертификата и ключа по умолчанию.
const (
	certDir      = "cert"
	certFileName = "server.crt"
	keyFileName  = "server.key"
)

// CertPaths — пути к файлам сертификата и приватного ключа.
type CertPaths struct {
	// CertPath — путь к PEM-файлу сертификата.
	CertPath string
	// KeyPath — путь к PEM-файлу приватного ключа.
	KeyPath string
}

// ProvideCertAndKey возвращает пути к самоподписным сертификату и ключу.
//
// Если файлы уже существуют, возвращает их пути без перегенерации. Иначе
// генерирует новую пару сертификат/ключ и записывает их на диск.
func ProvideCertAndKey() (paths CertPaths, err error) {
	paths = CertPaths{
		CertPath: filepath.Join(certDir, certFileName),
		KeyPath:  filepath.Join(certDir, keyFileName),
	}

	if fileExists(paths.CertPath) && fileExists(paths.KeyPath) {
		return paths, nil
	}

	if err = os.MkdirAll(certDir, 0755); err != nil {
		return CertPaths{}, err
	}

	defer func() {
		if err != nil {
			os.Remove(paths.CertPath)
			os.Remove(paths.KeyPath)
		}
	}()

	if err := generate(paths); err != nil {
		return CertPaths{}, err
	}

	return paths, nil
}

// generate создаёт самоподписный сертификат и записывает его и ключ на диск.
func generate(paths CertPaths) error {

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	certFile, err := os.Create(paths.CertPath)
	if err != nil {
		return err
	}
	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err != nil {
		certFile.Close()
		return err
	}
	err = certFile.Close()
	if err != nil {
		return err
	}

	keyFile, err := os.Create(paths.KeyPath)
	if err != nil {
		return err
	}
	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		keyFile.Close()
		return err
	}
	err = pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err != nil {
		keyFile.Close()
		return err
	}
	err = keyFile.Close()
	if err != nil {
		return err
	}

	return nil
}

// fileExists проверяет наличие файла на диске.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
