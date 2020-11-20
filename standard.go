// Copyright 2019 The Antilog Authors.
//
// SPDX-License-Identifier: MIT

package ulog

import (
	"context"
	"io"
	"io/ioutil"
)

var uLog = New()

// WithWriter returns a copy of the standard ULog instance configured to write to the given writer
func WithWriter(w io.Writer) ULog {
	return ULog{Writer: w, MessageKey: DefaultMessageKey, TimestampKey: DefaultTimestampKey}
}

// With returns a copy of the standard ULog instance configured with the provided fields
func With(fields ...Field) ULog {
	return uLog.With(fields...)
}

// WithKeyNames returns a copy of the ULog instance with the provided key names for timestamp and message keys.
func WithKeyNames(timestampKey, messageKey string) ULog {
	return uLog.WithKeyNames(timestampKey, messageKey)
}

// Write a message using the standard ULog instance
func Write(msg string, fields ...Field) {
	uLog.Write(msg, fields...)
}

// Log is the same as go-kit/kit/log.Log.
func Log(keyvals ...interface{}) error {
	return uLog.Log(keyvals...)
}

// WithContext returns a Context, storing the default ULog in it.
func WithContext(ctx context.Context) context.Context {
	return uLog.WithContext(ctx)
}

// WithContext returns a Context, storing the ULog in int.
func (u ULog) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, logCtxKey, uLog)
}

// FromContext returns the ULog from the Context,
// or a disabled logger if no logger is set on the Context.
func FromContext(ctx context.Context) ULog {
	if I := ctx.Value(logCtxKey); I != nil {
		if lgr, ok := I.(ULog); ok {
			return lgr
		}
	}
	return ULog{Writer: ioutil.Discard}
}

type ctxKey string

const logCtxKey = ctxKey("ULog")
