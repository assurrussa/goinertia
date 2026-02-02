package inertiat

import (
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/mock"
)

// MockSessionStore is a reusable testify-based mock for inertia.SessionStore used across tests.
type MockSessionStore struct {
	mock.Mock
}

func (m *MockSessionStore) Get(c fiber.Ctx, key string) (any, error) {
	args := m.Called(c, key)
	return args.Get(0), args.Error(1)
}

func (m *MockSessionStore) Set(c fiber.Ctx, key string, value any) error {
	args := m.Called(c, key, value)
	return args.Error(0)
}

func (m *MockSessionStore) Delete(c fiber.Ctx, key string) error {
	args := m.Called(c, key)
	return args.Error(0)
}

func (m *MockSessionStore) Flash(c fiber.Ctx, key string, value any) error {
	args := m.Called(c, key, value)
	return args.Error(0)
}

func (m *MockSessionStore) GetFlash(c fiber.Ctx, key string) (any, error) {
	args := m.Called(c, key)
	return args.Get(0), args.Error(1)
}
