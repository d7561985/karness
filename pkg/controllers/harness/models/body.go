package models

import (
	"bytes"
	"fmt"
	"github.com/d7561985/karness/pkg/apis/karness/v1alpha1"
	"k8s.io/apimachinery/pkg/util/json"
	"sync"
	"text/template"
)

func Body(in *v1alpha1.Body) *body {
	return &body{Body: in}
}

type body struct {
	*v1alpha1.Body
}

func (b *body) Get() ([]byte, error) {
	if len(b.Byte) > 0 {
		return b.Byte, nil
	}

	if len(b.JSON) > 0 {
		return []byte(b.JSON), nil
	}

	if len(b.KV) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(b.KV)
	if err != nil {
		return nil, fmt.Errorf("kv marshal error: %w", err)
	}

	return body, nil
}

func (b *body) GetBody(store *sync.Map) ([]byte, error) {
	res, err := b.Get()
	if err != nil {
		return nil, err
	}

	sm := make(map[string]interface{})
	store.Range(func(key, value interface{}) bool {
		sm[key.(string)] = value
		return true
	})

	t, err := template.New("x").Parse(string(res))
	if err != nil {
		return nil, fmt.Errorf("template parse error: %w", err)
	}

	buf := bytes.NewBuffer(nil)
	if err = t.Execute(buf, sm); err != nil {
		return nil, fmt.Errorf("store template executor: %w", err)
	}

	return buf.Bytes(), nil
}
