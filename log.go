// Copyright 2020 Tamás Gulácsi.
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
	"io"
	"os"
	"sync"
	"time"
)

// ULog is the antidote to modern loggers
type ULog struct {
	Writer io.Writer

	fields                   encodedFields
	TimestampKey, MessageKey string `json:"-"`
}

// New instance of ULog
func New() ULog {
	return ULog{TimestampKey: DefaultTimestampKey, MessageKey: DefaultMessageKey}
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

var (
	scratchBuffers = sync.Pool{New: func() interface{} { x := make([]byte, 0, 1024); return bytes.NewBuffer(x) }}
	scratchFields  = sync.Pool{New: func() interface{} { var x encodedFields; return &x }}
)

const (
	DefaultTimestampKey = "ts"
	DefaultMessageKey   = "msg"
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
	sb.Grow(3 + len(tsKey) + 4 + len(time.RFC3339) + 4 + len(msgKey) + 3 + 1 + len(msg) + 1 + fieldsLen + 2)
	sb.WriteString(`{ "`)
	sb.WriteString(tsKey)
	sb.WriteString(`": "`)
	sb.WriteString(now.Format(time.RFC3339))
	sb.WriteString(`", "`)
	sb.WriteString(msgKey)
	sb.WriteString(`": `)
	_ = json.NewEncoder(sb).Encode(msg)

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
	sb.WriteString(` }`)

	w := u.Writer
	if w == nil {
		w = os.Stderr
	}
	_, _ = w.Write(sb.Bytes())

	scratchFields.Put(eF.Reset())
	sb.Reset()
	scratchBuffers.Put(sb)
}

func toJSON(field Field) string {
	// In the case of errors, explicitly destructure them
	if err, ok := field.(error); ok {
		field = err.Error()
	}

	// For anything else, just let json.Marshal do it
	bytes, err := json.Marshal(field)
	if err != nil {
		return string(err.Error())
	}

	return string(bytes)
}
