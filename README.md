> ## ⚠️ Work in Progress
>
> This package is still work in progress and is not recommended to be used.

# Tinyflags

[![Go Reference](https://pkg.go.dev/badge/github.com/markusylisiurunen/go-tinyflags.svg)](https://pkg.go.dev/github.com/markusylisiurunen/go-tinyflags)

**Table of Contents**

1. [Install](#install)
2. [Usage](#usage)

## Install

```sh
go get github.com/markusylisiurunen/go-tinyflags
```

## Usage

```go
type CustomStruct struct {
  Enabled bool   `json:"enabled"`
  Value   string `json:"value"`
}

var manager = tinyflags.New(
  tinyflags.NewConstantStore().
    With("language", "en").
    With("reduced_motion", true).
    With("custom", CustomStruct{true, "hello world"}),
)

func ExampleSimpleRead(ctx context.Context) {
  var (
    languageFlag      = tinyflags.NewStringFlag("language")
    reducedMotionFlag = tinyflags.NewBoolFlag("reduced_motion")
  )
  if err := manager.Read(ctx, &languageFlag, &reducedMotionFlag); err != nil {
    fmt.Printf("error reading flags: %v\n", err)
    return
  }
  fmt.Printf("language: %s\n", languageFlag.Get())
  fmt.Printf("reduced motion: %t\n", reducedMotionFlag.Get())
}

func ExampleSimpleWrite(ctx context.Context) {
  var (
    languageFlag      = tinyflags.NewStringFlag("language").With("en")
    reducedMotionFlag = tinyflags.NewBoolFlag("reduced_motion").With(true)
  )
  if err := manager.Write(ctx, &languageFlag, &reducedMotionFlag); err != nil {
    fmt.Printf("error writing flags: %v\n", err)
    return
  }
}

func ExampleCustomStruct(ctx context.Context) {
  var (
    customFlag = tinyflags.NewFlag[CustomStruct]("custom")
  )
  if err := manager.Read(ctx, &customFlag); err != nil {
    fmt.Printf("error reading flag: %v\n", err)
    return
  }
  fmt.Printf("enabled: %t\n", customFlag.Get().Enabled)
  fmt.Printf("value: %s\n", customFlag.Get().Value)
}
```
