package main

import "context"

func main() {
	ctx := context.Background()
	SimpleRead(ctx)
	SimpleWrite(ctx)
	StructReadWrite(ctx)
}
