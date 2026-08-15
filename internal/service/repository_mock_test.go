package service

import (
	"context"

	"github.com/victor/gophkeeper/internal/model"
	repoerrors "github.com/victor/gophkeeper/internal/repository/errors"
)

// mockRepo — in-memory реализация repository.Repository для тестов.
type mockRepo struct {
	users    map[string]*model.User // key: login
	objects  map[string]*model.Object
	metadata map[string]*model.Metadata
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		users:    map[string]*model.User{},
		objects:  map[string]*model.Object{},
		metadata: map[string]*model.Metadata{},
	}
}

func (m *mockRepo) Close() error { return nil }

func (m *mockRepo) CreateUser(_ context.Context, u *model.User) error {
	if _, ok := m.users[u.Login]; ok {
		return repoerrors.ErrLoginExists
	}
	m.users[u.Login] = u
	return nil
}

func (m *mockRepo) GetUserByLogin(_ context.Context, login string) (*model.User, error) {
	u, ok := m.users[login]
	if !ok {
		return nil, repoerrors.ErrNotFound
	}
	return u, nil
}

func (m *mockRepo) CreateObject(_ context.Context, o *model.Object) error {
	m.objects[o.ID] = o
	return nil
}

func (m *mockRepo) GetObject(_ context.Context, userID, id string) (*model.Object, error) {
	o, ok := m.objects[id]
	if !ok || o.UserID != userID {
		return nil, repoerrors.ErrNotFound
	}
	return o, nil
}

func (m *mockRepo) ListObjects(_ context.Context, userID string) ([]*model.Object, error) {
	result := []*model.Object{}
	for _, o := range m.objects {
		if o.UserID == userID {
			result = append(result, o)
		}
	}
	return result, nil
}

func (m *mockRepo) UpdateObject(_ context.Context, o *model.Object) error {
	if _, ok := m.objects[o.ID]; !ok {
		return repoerrors.ErrNotFound
	}
	m.objects[o.ID] = o
	return nil
}

func (m *mockRepo) DeleteObject(_ context.Context, userID, id string) error {
	o, ok := m.objects[id]
	if !ok || o.UserID != userID {
		return repoerrors.ErrNotFound
	}
	delete(m.objects, id)
	return nil
}

func (m *mockRepo) CreateMetadata(_ context.Context, md *model.Metadata) error {
	m.metadata[md.ID] = md
	return nil
}

func (m *mockRepo) GetMetadata(_ context.Context, userID, objectID, metaID string) (*model.Metadata, error) {
	md, ok := m.metadata[metaID]
	if !ok || md.UserID != userID || md.ObjectID != objectID {
		return nil, repoerrors.ErrNotFound
	}
	return md, nil
}

func (m *mockRepo) ListMetadata(_ context.Context, userID, objectID string) ([]*model.Metadata, error) {
	result := []*model.Metadata{}
	for _, md := range m.metadata {
		if md.UserID == userID && md.ObjectID == objectID {
			result = append(result, md)
		}
	}
	return result, nil
}

func (m *mockRepo) UpdateMetadata(_ context.Context, md *model.Metadata) error {
	if _, ok := m.metadata[md.ID]; !ok {
		return repoerrors.ErrNotFound
	}
	m.metadata[md.ID] = md
	return nil
}

func (m *mockRepo) DeleteMetadata(_ context.Context, userID, objectID, metaID string) error {
	md, ok := m.metadata[metaID]
	if !ok || md.UserID != userID || md.ObjectID != objectID {
		return repoerrors.ErrNotFound
	}
	delete(m.metadata, metaID)
	return nil
}
