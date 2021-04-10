package checker

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/d7561985/karness/pkg/apis/karness/v1alpha1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/klog/v2"
)

type ResCheck v1alpha1.ConditionResponse

func (r ResCheck) Is(status string, res []byte) bool {
	if r.Status != "" {
		if r.Status != status {
			fmt.Println(">>1")
			return false
		}
	}

	if r.Body.JSON != nil {
		return *r.Body.JSON == string(res)
	}

	if len(r.Body.Byte) > 0 {
		return bytes.Equal(r.Body.Byte, res)
	}

	if len(r.Body.KV) > 0 {
		m := make(map[string]v1alpha1.Any)
		if err := json.Unmarshal(res, &m); err != nil {
			klog.Errorf("result (%s) cant' unmarshal to map", string(res))
			return false
		}

		return reflect.DeepEqual(r.Body.KV, m)
	}

	return true
}
