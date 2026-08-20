package tlscert

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"testing"
)

func TestProvideCertAndKey(t *testing.T) {
	t.Chdir(t.TempDir())

	paths, err := ProvideCertAndKey()
	if err != nil {
		t.Fatalf("ProvideCertAndKey: %v", err)
	}

	for _, p := range []string{paths.CertPath, paths.KeyPath} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("файл %s не найден: %v", p, err)
		}
	}

	// Сертификат должен загружаться как валидная пара TLS.
	cert, err := tls.LoadX509KeyPair(paths.CertPath, paths.KeyPath)
	if err != nil {
		t.Fatalf("LoadX509KeyPair: %v", err)
	}
	if _, err := x509.ParseCertificate(cert.Certificate[0]); err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
}

func TestProvideCertAndKeyReusesExisting(t *testing.T) {
	t.Chdir(t.TempDir())

	first, err := ProvideCertAndKey()
	if err != nil {
		t.Fatalf("ProvideCertAndKey: %v", err)
	}

	firstCert, _ := os.ReadFile(first.CertPath)

	second, err := ProvideCertAndKey()
	if err != nil {
		t.Fatalf("ProvideCertAndKey: %v", err)
	}

	secondCert, _ := os.ReadFile(second.CertPath)
	if string(firstCert) != string(secondCert) {
		t.Fatal("повторный вызов не должен перегенерировать сертификат")
	}
}
