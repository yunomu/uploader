package cdn

import "context"

type CDN interface {
	Invalidate(ctx context.Context, paths []string) error
}
