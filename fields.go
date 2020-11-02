// Copyright 2020 Tamás Gulácsi.
// Copyright 2019 The Antilog Authors.
//
// SPDX-License-Identifier: MIT

package ulog

import (
	"bytes"
	"encoding/json"
	"sync"
)

// Field type for all inputs
type Field interface{}

// EncodedField type for storing fields in after conversion to JSON
type encodedField [2]string

// Key of the encoded field
func (f encodedField) Key() string {
	return f[0]
}

// Value of the encoded field
func (f encodedField) Value() string {
	return f[1]
}

// encodedFields is a list of encoded fields
type encodedFields []encodedField

// Add and encode fields.
func (eF *encodedFields) AppendFields(fields []Field) *encodedFields {
	if eF == nil {
		return eF
	}
	eF.Grow(len(fields) / 2)
	js := scratchJS.Get().(*jsonEncoder)
	for ix := 0; ix < len(fields); ix += 2 {
		rawKey := fields[ix]
		rawValue := fields[ix+1]

		keyString, ok := rawKey.(string)
		if !ok {
			continue
		}

		key := js.JSON(keyString)
		value := js.JSON(rawValue)

		if i := eF.Index(key); i >= 0 {
			(*eF)[i][1] = value
			continue
		}

		*eF = append(*eF, encodedField{key, value})
	}
	scratchJS.Put(js)
	return eF
}

// AppendUnique encoded field if the key is not already set
func (eF *encodedFields) AppendEncoded(fields encodedFields) *encodedFields {
	if eF == nil {
		return eF
	}
	eF.Grow(len(fields))
	for _, f := range fields {
		if i := eF.Index(f.Key()); i >= 0 {
			(*eF)[i][1] = f.Value()
		} else {
			*eF = append(*eF, f)
		}
	}
	return eF
}

func (eF *encodedFields) Index(key string) int {
	if eF == nil {
		return -1
	}
	for i, v := range *eF {
		if v.Key() == key {
			return i
		}
	}
	return -1
}

func (eF *encodedFields) Grow(length int) *encodedFields {
	if len(*eF)+length > cap(*eF) {
		x := make([]encodedField, len(*eF), len(*eF)+length)
		copy(x, *eF)
		*eF = x
	}
	return eF
}

func (eF *encodedFields) Reset() *encodedFields { *eF = (*eF)[:0]; return eF }

type jsonEncoder struct {
	buf *bytes.Buffer
	enc *json.Encoder
}

var scratchJS = sync.Pool{New: func() interface{} {
	js := jsonEncoder{buf: scratchBuffers.Get().(*bytes.Buffer)}
	js.buf.Reset()
	js.enc = json.NewEncoder(js.buf)
	return &js
}}

func (js *jsonEncoder) JSON(v interface{}) string {
	if err, ok := v.(error); ok {
		v = err.Error()
	}
	js.buf.Reset()
	if err := js.enc.Encode(v); err != nil {
		return err.Error()
	}
	b := js.buf.Bytes()
	if len(b) == 0 {
		return ""
	}
	if b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return string(b)
}
