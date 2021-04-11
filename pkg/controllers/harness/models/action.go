package models

import (
	"github.com/d7561985/karness/pkg/apis/karness/v1alpha1"
	"sync"
)

type Action struct {
	v1alpha1.Action
}

func NewAction(a v1alpha1.Action) *Action {
	return &Action{Action: a}
}

// GetBody take stora sync.Map and fill
func (a *Action) GetBody(store *sync.Map) ([]byte, error) {
	return Body(&a.Body).GetBody(store)
}
