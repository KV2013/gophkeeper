package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/victor/gophkeeper/internal/model"
)

func TestClient(t *testing.T) {
	tests := map[string]struct {
		handler http.HandlerFunc
		act     func(c *Client) error
	}{
		"register возвращает токен и соль": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					Login    string `json:"login"`
					Password string `json:"password"`
				}
				_ = json.NewDecoder(r.Body).Decode(&req)
				if req.Login != "bob" || req.Password != "hunter2" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"token": "tok", "salt": "c2FsdA=="})
			},
			act: func(c *Client) error {
				resp, err := c.Register(context.Background(), "bob", "hunter2")
				if err != nil {
					return err
				}
				if resp.Token != "tok" {
					return fmt.Errorf("Token: got %q, want tok", resp.Token)
				}
				if string(resp.Salt) != "salt" {
					return fmt.Errorf("Salt: got %q, want salt", resp.Salt)
				}
				return nil
			},
		},
		"create object отправляет заголовок Authorization": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer my-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				writeJSON(w, http.StatusOK, model.Object{ID: "id-1", Name: "test", Type: model.SecretTypeText})
			},
			act: func(c *Client) error {
				obj, err := c.CreateObject(context.Background(), "my-token", CreateObjectRequest{
					Name:       "test",
					Type:       model.SecretTypeText,
					Salt:       []byte("0123456789abcdef"),
					Ciphertext: []byte("cipher"),
				})
				if err != nil {
					return err
				}
				if obj.ID != "id-1" {
					return fmt.Errorf("ID: got %q, want id-1", obj.ID)
				}
				return nil
			},
		},
		"list objects возвращает список": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, []*model.Object{
					{ID: "a", Name: "one"},
					{ID: "b", Name: "two"},
				})
			},
			act: func(c *Client) error {
				objects, err := c.ListObjects(context.Background(), "tok")
				if err != nil {
					return err
				}
				if len(objects) != 2 {
					return fmt.Errorf("got %d, want 2", len(objects))
				}
				return nil
			},
		},
		"ошибка сервера превращается в *Error": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "объект не найден"})
			},
			act: func(c *Client) error {
				_, err := c.GetObject(context.Background(), "tok", "missing")
				if err == nil {
					return fmt.Errorf("ожидалась ошибка")
				}
				var apiErr *Error
				if !errors.As(err, &apiErr) {
					return fmt.Errorf("ожидалась *api.Error, got %T", err)
				}
				if apiErr.StatusCode != http.StatusNotFound {
					return fmt.Errorf("StatusCode: got %d, want 404", apiErr.StatusCode)
				}
				return nil
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()

			client, err := New(srv.URL)
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if err := tc.act(client); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func TestTLS(t *testing.T) {
	certPEM, cert := selfSignedCert(t)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"ok": "1"})
	}))
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	srv.StartTLS()
	defer srv.Close()

	caFile := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caFile, certPEM, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	tests := map[string]struct {
		opts    []Option
		wantErr bool
	}{
		"с CA-сертификатом подключение проходит": {
			opts: []Option{WithCACertFile(caFile)},
		},
		"без CA-сертификата подключение не проходит": {
			wantErr: true,
		},
		"с несуществующим CA-файлом": {
			opts:    []Option{WithCACertFile("/nonexistent/ca.pem")},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client, err := New(srv.URL, tc.opts...)
			if err != nil {
				if tc.wantErr {
					return
				}
				t.Fatalf("New: %v", err)
			}

			_, err = client.do(context.Background(), http.MethodGet, "/", "", nil)
			if tc.wantErr {
				if err == nil {
					t.Fatal("ожидалась ошибка")
				}
				return
			}
			if err != nil {
				t.Fatalf("do: %v", err)
			}
		})
	}
}

func selfSignedCert(t *testing.T) ([]byte, tls.Certificate) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair: %v", err)
	}
	return certPEM, cert
}
