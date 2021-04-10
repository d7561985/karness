package harness

import "context"

type Processor interface {
	// Start should be blocking operation
	Start(ctx context.Context)
}
