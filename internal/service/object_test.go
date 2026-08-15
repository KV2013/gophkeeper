package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"github.com/victor/gophkeeper/internal/crypto"
	"github.com/victor/gophkeeper/internal/model"
)

func newTestObjectService() (*ObjectService, *mockRepo) {
	repo := newMockRepo()
	return NewObjectService(repo, zap.NewNop()), repo
}

func testSalt(t *testing.T) []byte {
	t.Helper()
	salt, err := crypto.NewSalt()
	if err != nil {
		t.Fatalf("NewSalt: %v", err)
	}
	return salt
}

func TestCreateObjectSuccess(t *testing.T) {
	svc, _ := newTestObjectService()

	obj, err := svc.CreateObject(context.Background(), "user-1", model.SecretTypeText, testSalt(t), []byte("cipher"))
	if err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	if obj.ID == "" {
		t.Fatal("ожидался непустой ID объекта")
	}
	if obj.UserID != "user-1" {
		t.Fatalf("UserID: got %q, want user-1", obj.UserID)
	}
}

func TestCreateObjectInvalidType(t *testing.T) {
	svc, _ := newTestObjectService()

	_, err := svc.CreateObject(context.Background(), "user-1", model.SecretType("bogus"), testSalt(t), []byte("cipher"))
	if !errors.Is(err, ErrBadRequest) {
		t.Fatalf("ожидалась ErrBadRequest, got %v", err)
	}
}

func TestCreateObjectInvalidSalt(t *testing.T) {
	svc, _ := newTestObjectService()

	_, err := svc.CreateObject(context.Background(), "user-1", model.SecretTypeText, []byte("short"), []byte("cipher"))
	if !errors.Is(err, ErrBadRequest) {
		t.Fatalf("ожидалась ErrBadRequest, got %v", err)
	}
}

func TestGetObjectNotFound(t *testing.T) {
	svc, _ := newTestObjectService()

	if _, err := svc.GetObject(context.Background(), "user-1", "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ожидалась ErrNotFound, got %v", err)
	}
}

func TestUpdateObject(t *testing.T) {
	svc, _ := newTestObjectService()

	obj, err := svc.CreateObject(context.Background(), "user-1", model.SecretTypeText, testSalt(t), []byte("cipher"))
	if err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	updated, err := svc.UpdateObject(context.Background(), "user-1", obj.ID, model.SecretTypeCard, testSalt(t), []byte("new-cipher"))
	if err != nil {
		t.Fatalf("UpdateObject: %v", err)
	}
	if updated.Type != model.SecretTypeCard {
		t.Fatalf("Type: got %q, want card", updated.Type)
	}
	if string(updated.Ciphertext) != "new-cipher" {
		t.Fatalf("Ciphertext: got %q, want new-cipher", updated.Ciphertext)
	}
}

func TestDeleteObject(t *testing.T) {
	svc, _ := newTestObjectService()

	obj, err := svc.CreateObject(context.Background(), "user-1", model.SecretTypeText, testSalt(t), []byte("cipher"))
	if err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	if err := svc.DeleteObject(context.Background(), "user-1", obj.ID); err != nil {
		t.Fatalf("DeleteObject: %v", err)
	}
	if err := svc.DeleteObject(context.Background(), "user-1", obj.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ожидалась ErrNotFound при повторном удалении, got %v", err)
	}
}

func TestCreateMetadata(t *testing.T) {
	svc, _ := newTestObjectService()

	obj, err := svc.CreateObject(context.Background(), "user-1", model.SecretTypeText, testSalt(t), []byte("cipher"))
	if err != nil {
		t.Fatalf("CreateObject: %v", err)
	}

	m, err := svc.CreateMetadata(context.Background(), "user-1", obj.ID, "website", 1, map[string]string{"url": "example.com"})
	if err != nil {
		t.Fatalf("CreateMetadata: %v", err)
	}
	if m.ID == "" {
		t.Fatal("ожидался непустой ID метаданных")
	}
	if m.Options["url"] != "example.com" {
		t.Fatalf("Options: got %v", m.Options)
	}
}

func TestCreateMetadataInvalidName(t *testing.T) {
	svc, _ := newTestObjectService()

	if _, err := svc.CreateMetadata(context.Background(), "user-1", "obj", "", 0, nil); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("ожидалась ErrBadRequest, got %v", err)
	}
}

func TestListMetadata(t *testing.T) {
	svc, _ := newTestObjectService()

	obj, err := svc.CreateObject(context.Background(), "user-1", model.SecretTypeText, testSalt(t), []byte("cipher"))
	if err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	if _, err := svc.CreateMetadata(context.Background(), "user-1", obj.ID, "a", 0, nil); err != nil {
		t.Fatalf("CreateMetadata: %v", err)
	}
	if _, err := svc.CreateMetadata(context.Background(), "user-1", obj.ID, "b", 1, nil); err != nil {
		t.Fatalf("CreateMetadata: %v", err)
	}

	list, err := svc.ListMetadata(context.Background(), "user-1", obj.ID)
	if err != nil {
		t.Fatalf("ListMetadata: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListMetadata: got %d записей, want 2", len(list))
	}
}

func TestUpdateAndDeleteMetadata(t *testing.T) {
	svc, _ := newTestObjectService()

	obj, err := svc.CreateObject(context.Background(), "user-1", model.SecretTypeText, testSalt(t), []byte("cipher"))
	if err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	m, err := svc.CreateMetadata(context.Background(), "user-1", obj.ID, "website", 0, nil)
	if err != nil {
		t.Fatalf("CreateMetadata: %v", err)
	}

	updated, err := svc.UpdateMetadata(context.Background(), "user-1", obj.ID, m.ID, "renamed", 2, map[string]string{"a": "b"})
	if err != nil {
		t.Fatalf("UpdateMetadata: %v", err)
	}
	if updated.Name != "renamed" || updated.OrderNumber != 2 {
		t.Fatalf("UpdateMetadata: got name=%q order=%d", updated.Name, updated.OrderNumber)
	}

	if err := svc.DeleteMetadata(context.Background(), "user-1", obj.ID, m.ID); err != nil {
		t.Fatalf("DeleteMetadata: %v", err)
	}
	if err := svc.DeleteMetadata(context.Background(), "user-1", obj.ID, m.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ожидалась ErrNotFound при повторном удалении, got %v", err)
	}
}
