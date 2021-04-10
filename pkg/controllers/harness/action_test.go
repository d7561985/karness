package harness

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/util/jsonpath"
	"reflect"
	"testing"
)

func TestActionResult_GetKeyValue(t *testing.T) {
	type fields struct {
		Code string
		Body []byte
	}

	tests := []struct {
		name    string
		fields  fields
		key     string
		want    string
		wantErr error
	}{
		{
			"has json",
			fields{Body: []byte(`{"message": 1}`)},
			"{.message}",
			"1",
			nil,
		},
		{
			"bad json path",
			fields{Body: []byte(`{"message": 1}`)},
			"message",
			"",
			ErrBadJsonPath,
		},
		{
			"bad path",
			fields{Body: []byte(`{"message": 1}`)},
			"{.QQQ}",
			"",
			ErrNoKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ActionResult{
				Code: tt.fields.Code,
				Body: tt.fields.Body,
			}
			got, err := a.GetKeyValue(tt.key)
			fmt.Println(got)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr))
			}

			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("GetKeyValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetKeyValue() got = %v(%[1]T), want %v(%[2]T)", got, tt.want)
			}
		})
	}
}

// https://github.com/kubernetes/kubernetes/blob/758c56cc85e554122602d233cb315a07dd0e6961/pkg/util/jsonpath/jsonpath_test.go
func TestJsonPath(t *testing.T) {
	jp := jsonpath.New("x")
	assert.NoError(t, jp.Parse("{.message}"))
	buf := bytes.NewBuffer(nil)

	err := jp.Execute(buf, map[string]interface{}{"message": 1})
	assert.NoError(t, err)
	assert.Equal(t, "1", buf.String())
}
