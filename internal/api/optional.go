package api

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
)

type Optional[T any] struct {
	Value *T
	Set   bool
}

func (o *Optional[T]) UnmarshalJSONFrom(decoder *jsontext.Decoder) error {
	o.Set = true
	return json.UnmarshalDecode(decoder, &o.Value)
}
