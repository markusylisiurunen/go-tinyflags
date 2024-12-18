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
	flags := tinyflags.New(
		tinyflags.NewConstantStore().
			With("custom", CustomStruct{true, "hello world"}),
	)
	defer flags.Close()
	var (
		customFlag = tinyflags.NewFlag[CustomStruct]("custom")
	)
	if err := flags.Read(ctx, &customFlag); err != nil {
		fmt.Printf("error reading flags: %v\n", err)
		return
	}
	fmt.Printf("enabled: %t\n", customFlag.Get().Enabled)
	fmt.Printf("value: %s\n", customFlag.Get().Value)
}
