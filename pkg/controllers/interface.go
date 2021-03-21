package controllers

import (
	"context"

	api "github.com/d7561985/karness/pkg/apis/karness/v1alpha1"
)

type Kube interface {
	Update(item *api.Scenario) error
}

type HarnessFactory interface {
	// Add should be called via Kube controller when object created
	Add(ctx context.Context, add interface{})
}
