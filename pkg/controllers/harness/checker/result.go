package checker

import (
	"bytes"
	"fmt"
	"github.com/d7561985/karness/pkg/controllers/harness/models"
	"sync"

	"github.com/d7561985/karness/pkg/apis/karness/v1alpha1"
)

type ResCheckInterface interface {
	Is(status string, res []byte) error
}

type rs struct {
	store     *sync.Map
	condition *v1alpha1.ConditionResponse
}

func ResCheck(store *sync.Map, condition *v1alpha1.ConditionResponse) ResCheckInterface {
	return &rs{
		store:     store,
		condition: condition,
	}
}

func (r *rs) Is(status string, resBody []byte) error {
	if r.condition.Status != "" {
		if r.condition.Status != status {
			return fmt.Errorf("bad status %q expect %q", status, r.condition.Status)
		}
	}

	body, err := models.Body(&r.condition.Body).GetBody(r.store)
	if err != nil {
		return fmt.Errorf("get condition body %w", err)
	}

	if bytes.Equal(body, resBody) {
		return nil
	}

	return fmt.Errorf("condition %q not equal result %q", string(body), string(resBody))
}
