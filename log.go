// Copyright 2020, 2021 Tamás Gulácsi.
// Copyright 2019 The Antilog Authors.
//
// SPDX-License-Identifier: MIT

// Package ulog is the antidote to modern loggers.
//
// ulog only logs JSON formatted output. Structured logging is the only good logging.
//
// ulog does not have log levels. If you don't want something logged, don't log it.
//
// ulog does support setting fields in context.
// Useful for building a log context over the course of an operation.
package ulog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
	"unsafe"
)

// ULog is the antidote to modern loggers
type ULog struct {
	Writer                   io.Writer
	TimestampKey, MessageKey string `json:"-"`

	fields encodedFields
}

// New instance of ULog
func New() ULog {
	return ULog{TimestampKey: DefaultTimestampKey, MessageKey: DefaultMessageKey, Writer: DefaultWriter}
}

// With returns a copy of the ULog instance with the provided fields preset for every subsequent call.
func (u ULog) With(fields ...Field) ULog {
	v := u
	ff := scratchFields.Get().(*encodedFields).
		Reset().
		Grow(len(fields) + len(v.fields))
	v.fields = *ff.AppendEncoded(v.fields).AppendFields(fields)
	return v
}

// WithKeyNames returns a copy of the ULog instance with the provided key names for timestamp and message keys.
func (u ULog) WithKeyNames(timestampKey, messageKey string) ULog {
	v := u
	if timestampKey == "" {
		timestampKey = DefaultTimestampKey
	}
	if messageKey == "" {
		messageKey = DefaultMessageKey
	}
	v.TimestampKey, v.MessageKey = timestampKey, messageKey
	return v
}

// Log makes ULog implement github.com/go-kit/kit/log.Logger.
func (u ULog) Log(keyvals ...interface{}) error {
	if len(keyvals) == 0 {
		return nil
	}
	if u.MessageKey == "" {
		u.MessageKey = DefaultMessageKey
	}
	var ok bool
	if len(keyvals)%2 != 0 {
		if _, ok = keyvals[0].(string); !ok {
			keyvals[0] = fmt.Sprintf("%v", keyvals[0])
		}
		u.Log(append(append(make([]interface{}, 0, 1+len(keyvals)), u.MessageKey), keyvals...)...)
		return nil
	}

	fields := *((*[]Field)(unsafe.Pointer(&keyvals)))
	if u.TimestampKey == "" {
		u.TimestampKey = DefaultTimestampKey
	}
	var msg string
	for i := 0; i < len(fields); i += 2 {
		s, ok := fields[i].(string)
		if ok {
			if s == u.MessageKey && msg == "" {
				if msg, ok = fields[i+1].(string); !ok {
					msg = fmt.Sprintf("%v", keyvals[i+1])
				}
				fields[i], fields[i+1] = fields[0], fields[1]
				fields = fields[2:]
				i -= 2
			} else if s == u.TimestampKey {
				if _, ok = fields[i+1].(time.Time); ok {
					fields[i], fields[i+1] = fields[0], fields[1]
					fields = fields[2:]
					i -= 2
				}
			}
		}
	}

	u.Write(msg, fields...)
	return nil
}

var (
	scratchBuffers = sync.Pool{New: func() interface{} { x := make([]byte, 0, 1024); return bytes.NewBuffer(x) }}
	scratchFields  = sync.Pool{New: func() interface{} { var x encodedFields; return &x }}

	DefaultWriter = os.Stderr
)

const (
	DefaultTimestampKey = "ts"
	DefaultMessageKey   = "msg"

	timeFormat = "2006-01-02T15:04:05.999999"
)

// Write a JSON message to the configured writer or os.Stderr.
//
// Includes the message with the key `msg`. Includes the timestamp with the
// key `ts`. The timestamp field is always first and the message second.
//
// Fields in context will not be overridden. ULog will log the same key
// multiple times if it is set multiple times. If you don't want that, don't
// specify it multiple times.
func (u ULog) Write(msg string, fields ...Field) {
	now := time.Now().UTC()

	tsKey := u.TimestampKey
	if tsKey == "" {
		tsKey = DefaultTimestampKey
	}
	msgKey := u.MessageKey
	if msgKey == "" {
		msgKey = DefaultMessageKey
	}

	eF := scratchFields.Get().(*encodedFields).
		Reset().
		Grow(len(u.fields) + len(fields)/2).
		AppendEncoded(u.fields).AppendFields(fields)

	var fieldsLen int
	for _, field := range *eF {
		key := field.Key()
		if key == msgKey || key == tsKey {
			continue
		}
		fieldsLen += 2 + len(key) + 2 + len(field.Value())
	}

	sb := scratchBuffers.Get().(*bytes.Buffer)
	sb.Reset()
	sb.Grow(3 + len(tsKey) + 4 + len(timeFormat) + 5 + len(msgKey) + 3 + 1 + len(msg) + 1 + fieldsLen + 3)
	sb.WriteString(`{ "`)
	sb.WriteString(tsKey)
	sb.WriteString(`": "`)
	var a [len(timeFormat)]byte
	sb.Write(now.AppendFormat(a[:0], timeFormat))
	sb.WriteString(`Z", "`)
	sb.WriteString(msgKey)
	sb.WriteString(`": `)

	{
		n := sb.Len()
		enc := json.NewEncoder(sb)
		if err := enc.Encode(msg); err != nil {
			sb.Truncate(n)
			enc.Encode(fmt.Sprintf("%v", msg))
		}
	}
	if sb.Bytes()[sb.Len()-1] == '\n' {
		sb.Truncate(sb.Len() - 1)
	}

	for _, field := range *eF {
		key := field.Key()
		if key == msgKey || key == tsKey {
			continue
		}
		sb.WriteString(", ")
		sb.WriteString(key)
		sb.WriteString(`: `)
		sb.WriteString(field.Value())
	}
	sb.WriteString(" }\n")

	w := u.Writer
	if w == nil {
		w = DefaultWriter
	}
	_, _ = w.Write(sb.Bytes())

	scratchFields.Put(eF.Reset())
	sb.Reset()
	scratchBuffers.Put(sb)
}
