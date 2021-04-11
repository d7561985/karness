package models

import (
	"github.com/d7561985/karness/pkg/apis/karness/v1alpha1"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestAction_GetBody(t *testing.T) {
	tests := []struct {
		name string
		kv   map[string]interface{}
		body v1alpha1.Body
		want []byte
	}{
		{
			"OK JSON",
			map[string]interface{}{"MSG": "HELLO"},
			v1alpha1.Body{
				JSON: `{"KEY":"{{.MSG}}"}`,
			},
			[]byte(`{"KEY":"HELLO"}`),
		},
		{
			"OK BYTE",
			map[string]interface{}{"MSG": "HELLO"},
			v1alpha1.Body{
				Byte: []byte(`{"KEY":"{{.MSG}}"}`),
			},
			[]byte(`{"KEY":"HELLO"}`),
		},
		{
			"OK KV",
			map[string]interface{}{"MSG": "HELLO"},
			v1alpha1.Body{
				KV: map[string]v1alpha1.Any{"KEY": `{{.MSG}}`},
			},
			[]byte(`{"KEY":"HELLO"}`),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			act := NewAction(v1alpha1.Action{Body: test.body})

			store := sync.Map{}
			for k, v := range test.kv {
				store.Store(k, v)
			}

			res, err := act.GetBody(&store)
			assert.NoError(t, err)
			assert.Equal(t, test.want, res)
		})
	}
}
