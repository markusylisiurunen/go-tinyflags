package main

import (
	"context"
	"fmt"

	"github.com/markusylisiurunen/go-tinyflags"
)

type CustomStruct struct {
	Enabled bool   `json:"enabled"`
	Value   string `json:"value"`
}

func StructReadWrite(ctx context.Context) {
	manager := tinyflags.New(
		tinyflags.NewConstantStore().
			With("custom", CustomStruct{true, "hello world"}),
	)
	var (
		customFlag = tinyflags.NewFlag[CustomStruct]("custom")
	)
	if err := manager.Read(ctx, &customFlag); err != nil {
		fmt.Printf("error reading flags: %v\n", err)
		return
	}
	fmt.Printf("enabled: %t\n", customFlag.Get().Enabled)
	fmt.Printf("value: %s\n", customFlag.Get().Value)
}
