// Copyright 2020 Tamás Gulácsi.
// Copyright 2019 The Antilog Authors.
//
// SPDX-License-Identifier: MIT

package ulog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
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

type wrappedErr struct {
	Err, Details string
	err          error
}

func WrapError(err error) error {
	if err == nil {
		return nil
	}

	var pc [16]uintptr
	n := runtime.Callers(5, pc[:])
	var frames *runtime.Frames
	if n != 0 {
		frames = runtime.CallersFrames(pc[:n])
	}
	if frames == nil {
		return err
	}
	we := wrappedErr{err: err, Err: err.Error()}
	sb := scratchBuffers.Get().(*bytes.Buffer)
	sb.Reset()
	sb.WriteString(we.Err)
	// Loop to get frames.
	// A fixed number of pcs can expand to an indefinite number of Frames.
	for {
		frame, more := frames.Next()
		fmt.Fprintf(sb, "\n- %s:%d:%s", frame.File, frame.Line, frame.Function)
		if !more {
			break
		}
	}
	we.Details = sb.String()
	scratchBuffers.Put(sb)
	return &we
}

// StackTrace returns stack trace of an error.
func (we *wrappedErr) Error() string { return we.Err }
func (we *wrappedErr) Unwrap() error { return we.err }
func (we *wrappedErr) Format(f fmt.State, c rune) {
	if f.Flag('#') {
		fmt.Fprint(f, we.err)
	} else if f.Flag('+') {
		f.Write([]byte(we.Details))
	} else {
		f.Write([]byte(we.Err))
	}
}

func (js *jsonEncoder) JSON(v interface{}) string {
	if err, ok := v.(error); ok && err != nil {
		var we *wrappedErr
		if errors.As(err, &we) {
			v = we.Details
		} else {
			v = fmt.Sprintf("%+v", err)
		}
	}
	js.buf.Reset()
	if err := js.enc.Encode(v); err != nil {
		js.buf.Reset()
		js.enc.Encode(err.Error())
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
