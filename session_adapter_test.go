package goinertia_test

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/assurrussa/goinertia"
	inertiamocks "github.com/assurrussa/goinertia/mocks"
)

const (
	key1 = "key_1"
	val1 = "val_1"
)

func TestSessionAdapter_Get_Success(t *testing.T) {
	appCtx, adapter, mockAdapter, mockStore := createApp(t)
	mockAdapter.EXPECT().Get(appCtx).Return(mockStore, nil).Times(1)
	mockStore.EXPECT().Get(key1).Return(val1).Times(1)

	val, err := adapter.Get(appCtx, key1)
	require.NoError(t, err)
	require.Equal(t, val1, val)
}

func TestSessionAdapter_Get_Error(t *testing.T) {
	appCtx, adapter, mockAdapter, mockStore := createApp(t)
	mockAdapter.EXPECT().Get(appCtx).Return(mockStore, assert.AnError).Times(1)

	val, err := adapter.Get(appCtx, key1)
	require.ErrorIs(t, err, assert.AnError)
	require.Nil(t, val)
}

func TestSessionAdapter_Set_Success(t *testing.T) {
	appCtx, adapter, mockAdapter, mockStore := createApp(t)
	mockAdapter.EXPECT().Get(appCtx).Return(mockStore, nil).Times(1)
	mockStore.EXPECT().Set(key1, val1).Times(1)
	mockStore.EXPECT().Save().Return(nil).Times(1)

	err := adapter.Set(appCtx, key1, val1)
	require.NoError(t, err)
}

func TestSessionAdapter_Set_Error(t *testing.T) {
	appCtx, adapter, mockAdapter, mockStore := createApp(t)
	mockAdapter.EXPECT().Get(appCtx).Return(mockStore, assert.AnError).Times(1)

	err := adapter.Set(appCtx, key1, val1)
	require.ErrorIs(t, err, assert.AnError)
}

func TestSessionAdapter_Set_ErrorSave(t *testing.T) {
	appCtx, adapter, mockAdapter, mockStore := createApp(t)
	mockAdapter.EXPECT().Get(appCtx).Return(mockStore, nil).Times(1)
	mockStore.EXPECT().Set(key1, val1).Times(1)
	mockStore.EXPECT().Save().Return(assert.AnError).Times(1)

	err := adapter.Set(appCtx, key1, val1)
	require.ErrorIs(t, err, assert.AnError)
}

func TestSessionAdapter_Delete_Success(t *testing.T) {
	appCtx, adapter, mockAdapter, mockStore := createApp(t)
	mockAdapter.EXPECT().Get(appCtx).Return(mockStore, nil).Times(1)
	mockStore.EXPECT().Delete(key1).Times(1)
	mockStore.EXPECT().Save().Return(nil).Times(1)

	err := adapter.Delete(appCtx, key1)
	require.NoError(t, err)
}

func TestSessionAdapter_Delete_Error(t *testing.T) {
	appCtx, adapter, mockAdapter, mockStore := createApp(t)
	mockAdapter.EXPECT().Get(appCtx).Return(mockStore, assert.AnError).Times(1)

	err := adapter.Delete(appCtx, key1)
	require.ErrorIs(t, err, assert.AnError)
}

func TestSessionAdapter_Delete_ErrorSave(t *testing.T) {
	appCtx, adapter, mockAdapter, mockStore := createApp(t)
	mockAdapter.EXPECT().Get(appCtx).Return(mockStore, nil).Times(1)
	mockStore.EXPECT().Delete(key1).Times(1)
	mockStore.EXPECT().Save().Return(assert.AnError).Times(1)

	err := adapter.Delete(appCtx, key1)
	require.ErrorIs(t, err, assert.AnError)
}

func TestSessionAdapter_Flash_Success(t *testing.T) {
	appCtx, adapter, mockAdapter, mockStore := createApp(t)
	mockAdapter.EXPECT().Get(appCtx).Return(mockStore, nil).Times(1)
	mockStore.EXPECT().Set(key1, val1).Times(1)
	mockStore.EXPECT().Save().Return(nil).Times(1)

	err := adapter.Flash(appCtx, key1, val1)
	require.NoError(t, err)
}

func TestSessionAdapter_GetFlash_Success(t *testing.T) {
	appCtx, adapter, mockAdapter, mockStore := createApp(t)
	mockAdapter.EXPECT().Get(appCtx).Return(mockStore, nil).Times(1)
	mockStore.EXPECT().Get(key1).Return(val1).Times(1)
	mockStore.EXPECT().Delete(key1).Times(1)
	mockStore.EXPECT().Save().Return(nil).Times(1)

	val, err := adapter.GetFlash(appCtx, key1)
	require.NoError(t, err)
	require.Equal(t, val1, val)
}

func TestSessionAdapter_GetFlash_Error(t *testing.T) {
	appCtx, adapter, mockAdapter, mockStore := createApp(t)
	mockAdapter.EXPECT().Get(appCtx).Return(mockStore, assert.AnError).Times(1)

	val, err := adapter.GetFlash(appCtx, key1)
	require.ErrorIs(t, err, assert.AnError)
	require.Nil(t, val)
}

func TestSessionAdapter_GetFlash_ErrorSave(t *testing.T) {
	appCtx, adapter, mockAdapter, mockStore := createApp(t)
	mockAdapter.EXPECT().Get(appCtx).Return(mockStore, nil).Times(1)
	mockStore.EXPECT().Get(key1).Return(val1).Times(1)
	mockStore.EXPECT().Delete(key1).Times(1)
	mockStore.EXPECT().Save().Return(assert.AnError).Times(1)

	val, err := adapter.GetFlash(appCtx, key1)
	require.ErrorIs(t, err, assert.AnError)
	require.Nil(t, val)
}

func TestSessionAdapter_GetFlash_Empty(t *testing.T) {
	appCtx, adapter, mockAdapter, mockStore := createApp(t)
	mockAdapter.EXPECT().Get(appCtx).Return(mockStore, nil).Times(1)
	mockStore.EXPECT().Get(key1).Return(nil).Times(1)

	val, err := adapter.GetFlash(appCtx, key1)
	require.NoError(t, err)
	require.Nil(t, val)
}

func createApp(t *testing.T) (
	fiber.Ctx,
	*goinertia.FiberSessionAdapter[*inertiamocks.MockFiberSessionStore],
	*inertiamocks.MockSessionAdapter[*inertiamocks.MockFiberSessionStore],
	*inertiamocks.MockFiberSessionStore,
) {
	t.Helper()

	app := fiber.New()
	appCtx := fiber.NewDefaultCtx(app)
	ctrl := gomock.NewController(t)
	mockSessionStore := inertiamocks.NewMockFiberSessionStore(ctrl)
	mockSessionAdapter := inertiamocks.NewMockSessionAdapter[*inertiamocks.MockFiberSessionStore](ctrl)
	adapter := goinertia.NewFiberSessionAdapter[*inertiamocks.MockFiberSessionStore](mockSessionAdapter)

	return appCtx, adapter, mockSessionAdapter, mockSessionStore
}
