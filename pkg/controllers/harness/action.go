package harness

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/util/jsonpath"
)

type Action interface {
	Call(ctx context.Context) (*ActionResult, error)
}

var (
	ErrNoKey       = errors.New("provided key not exists in result")
	ErrBadJsonPath = errors.New("bad json path formula")
)

type ActionResult struct {
	Code string
	Body []byte
}

func OK() *ActionResult {
	return &ActionResult{Code: "OK"}
}

// GetKeyValue look up key in our json representation
// every time perform JSON marshaling. This is too slow!!!
func (a *ActionResult) GetKeyValue(jsonPath string) (string, error) {
	var tmp interface{}
	if err := json.Unmarshal(a.Body, &tmp); err != nil {
		return "", fmt.Errorf("can't unmarshal body: %w", err)
	}

	j := jsonpath.New(jsonPath)
	if err := j.Parse(jsonPath); err != nil {
		return "", fmt.Errorf("can't parse json path[%w]", err)
	}

	buf := bytes.NewBuffer(nil)
	if err := j.Execute(buf, tmp); err != nil {
		return "", fmt.Errorf("jsonpath execute error[%s]: %w", err, ErrNoKey)
	}

	if buf.String() == jsonPath {
		return "", ErrBadJsonPath
	}

	fmt.Println(">>>", buf.String())
	return buf.String(), nil
}
