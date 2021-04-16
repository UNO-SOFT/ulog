// Copyright 2021 Tamás Gulácsi.
//
// SPDX-License-Identifier: MIT

package ulog

func NewTestLogger(t interface{ Log(...interface{}) }) ULog {
	return ULog{TimestampKey: DefaultTimestampKey, MessageKey: DefaultMessageKey,
		Writer: testLogWriter(t.Log)}
}

type testLogWriter func(...interface{})

func (tw testLogWriter) Write(p []byte) (int, error) {
	tw(string(p))
	return len(p), nil
}
