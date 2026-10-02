package core

import "context"

//go:generate mockgen -source=$GOFILE -destination=../mocks/core_mock.gen.go -package=goinertiamocks

type Logger interface {
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

// SSRClient implementations must support concurrent Post calls and return an
// owned body. Reset is used only at shutdown/configuration, never during retry.
type SSRClient interface {
	Reset()
	Post(ctx context.Context, url string, body []byte, headers map[string]string) (int, []byte, error)
}
