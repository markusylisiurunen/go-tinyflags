package main

import (
	"context"
	"fmt"

	"github.com/markusylisiurunen/go-tinyflags"
)

func SimpleRead(ctx context.Context) {
	manager := tinyflags.New(
		tinyflags.NewConstantStore().
			With("language", "end").
			With("reduced_motion", true),
	)
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

func SimpleWrite(ctx context.Context) {
	manager := tinyflags.New(tinyflags.NewConstantStore())
	var (
		languageFlag      = tinyflags.NewStringFlag("language").With("en")
		reducedMotionFlag = tinyflags.NewBoolFlag("reduced_motion").With(true)
	)
	if err := manager.Write(ctx, &languageFlag, &reducedMotionFlag); err != nil {
		fmt.Printf("error writing flags: %v\n", err)
		return
	}
}