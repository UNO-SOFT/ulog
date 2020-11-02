# ulog

ulog is the antidote to modern loggers.

* ULog only logs JSON formatted output. Structured logging is the only good logging.
* ULog does not have log levels. If you don't want something logged, [don't log it](https://dave.cheney.net/2015/11/05/lets-talk-about-logging).
* ULog supports setting fields in context. This is useful for building a log context over the course of an operation.
* ULog has no dependencies. Using ulog only brings in what it needs, and that isn't much!
* ULog always uses RFC3339 formatted UTC timestamps, for sanity.

## Basic Usage

```go
    ulog.Write("a message")
```

```json
{ "ts": "2019-11-18T14:00:32Z", "msg": "a message" }
```

## With Fields

```go
    ulog.Write("a message",
        "field", "value",
        "a_number", 123,
        "a_bool", false,
    )
```

```json
{ "msg": "2019-11-18T14:00:32Z", "msg": "a message", "field": "value", "a_number": 123, "a_bool": false }
```

## With Context

```go
    logger := ulog.With(
        "request_id", "12345",
        "user_id": "big_jim_mcdonald",
    )

    logger.Write("a message",
        "field", "value",
        "a_number", 123,
        "a_bool", false)
```

```json
{ "ts": "2019-11-18T14:00:32Z", "msg": "a message", "request_id": "12345", "user_id": "big_jim_mcdonald", "field": "value", "a_number": 123, "a_bool": false }
```

## With More Complex Data Types

```go
    ulog.Write("something complex!",
        "array", []string{"this", "is", "an", "array"},
        "map", map[string]string{
            "key": "value",
            "just": "like that",
        },
        "the_ulog_struct_itself", ulog.With("hello", "world"),
    )
```

```json
{ "ts": "2019-11-18T13:41:56Z", "msg": "something complex!", "array": [ "this", "is", "an", "array" ], "map": { "key": "value", "just": "like that" }, "the_ulog_struct_itself": { "Fields": [ "hello", "world" ] } }
```

## Output To Somewhere Other Than STDOUT

```go
    var sb strings.Builder
    logger := ulog.WithWriter(sb)

    logger.Write("a message",
        "field", "value",
        "a_number", 123,
        "a_bool", false)

    fmt.Println(sb.String())
```

```json
{ "ts": "2019-11-18T14:00:32Z", "msg": "a message", "field": "value", "a_number": 123, "a_bool": false }
```
