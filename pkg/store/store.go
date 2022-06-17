package store

import (
	"context"
)

type Store interface {
	QueryRequisites(ctx context.Context, drvPaths ...string) ([]string, error)
}
