package main

import (
	"context"
	"fmt"
)

func main() {
	ctx := context.Background()
	fmt.Printf("simple read example:\n")
	SimpleRead(ctx)
	fmt.Printf("simple write example:\n")
	SimpleWrite(ctx)
	fmt.Printf("custom struct example:\n")
	StructReadWrite(ctx)
	fmt.Printf("full stack example:\n")
	StackReadWrite(ctx)
}
