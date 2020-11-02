// Copyright 2020 Tamás Gulácsi.
// Copyright 2019 The Antilog Authors.
//
// SPDX-License-Identifier: MIT

package ulog_test

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/UNO-SOFT/ulog"
)

var (
	fakeMessage = "Test logging, but use a somewhat realistic message length."
)

func BenchmarkLogEmpty(b *testing.B) {
	logger := ulog.New()
	logger.Writer = ioutil.Discard
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Write("")
		}
	})
}

func BenchmarkInfo(b *testing.B) {
	logger := ulog.WithWriter(ioutil.Discard)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Write(fakeMessage)
		}
	})
}

func BenchmarkContextFields(b *testing.B) {
	logger := ulog.WithWriter(ioutil.Discard).With(
		"string", "four!",
		"time", time.Time{},
		"int", 123,
		"float", -2.203230293249593)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Write(fakeMessage)
		}
	})
}

func BenchmarkContextAppend(b *testing.B) {
	logger := ulog.WithWriter(ioutil.Discard).With("foo", "bar")
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Write("", "bar", "baz")
		}
	})
}

func BenchmarkLogFields(b *testing.B) {
	logger := ulog.WithWriter(ioutil.Discard)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Write(fakeMessage,
				"string", "four!",
				"time", time.Time{},
				"int", 123,
				"float", -2.203230293249593)
		}
	})
}
