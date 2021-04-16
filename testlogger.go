// Copyright 2021 Tamás Gulácsi.
//
// SPDX-License-Identifier: MIT

package ulog

type testLogger interface {
	Log(...interface{})
}

func NewTestLogger(t testLogger) ULog {
	return ULog{TimestampKey: DefaultTimestampKey, MessageKey: DefaultMessageKey,
		Writer: testLogWriter{t}}
}

type testLogWriter struct {
	testLogger
}

func (tw testLogWriter) Write(p []byte) (int, error) {
	if helper, ok := tw.testLogger.(interface{ Helper() }); ok {
		helper.Helper()
	}
	tw.Log(string(p))
	return len(p), nil
}
