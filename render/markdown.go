package render

import (
	"github.com/grimdork/markulator"
)

func Markdown(src []byte) (string, error) {
	r := mk.NewRenderer(
		mk.WithSingleFile(true),
		mk.WithDoctype(false),
		mk.WithBodyTags(false),
	)
	v, err := r.RenderString(src)
	return string(v), err
}

