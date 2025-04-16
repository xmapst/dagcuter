package dagcuter

import "context"

type Task interface {
	Name() string
	Dependencies() []string

	PreExecution(ctx context.Context, input map[string]any) error

	Execute(ctx context.Context, input map[string]any) (map[string]any, error)

	PostExecution(ctx context.Context, output map[string]any) error
}
