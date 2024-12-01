package main

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/markusylisiurunen/go-tinyflags"
	"github.com/redis/go-redis/v9"

	_ "github.com/lib/pq"
)

func StackReadWrite(ctx context.Context) {
	tinyWait, longWait := 25*time.Millisecond, 250*time.Millisecond
	rateLimitFlagName := "rate_limit:" + fmt.Sprintf("%d", rand.Intn(1000))
	// connect to postgres
	postgresClient, err := sql.Open("postgres", "postgres://postgres:password@localhost:7402/dev?sslmode=disable")
	if err != nil {
		fmt.Printf("error connecting to postgres: %v\n", err)
		return
	}
	defer postgresClient.Close()
	if err := postgresClient.PingContext(ctx); err != nil {
		fmt.Printf("error pinging postgres: %v\n", err)
		return
	}
	// connect to redis
	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:7401"})
	defer redisClient.Close()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		fmt.Printf("error pinging redis: %v\n", err)
		return
	}
	// initialise the managers
	flags1 := tinyflags.New(
		tinyflags.NewMemoryStore(redisClient),
		tinyflags.NewRedisStore(redisClient, "example"),
		tinyflags.NewPostgresStore(postgresClient),
		tinyflags.NewConstantStore().With(rateLimitFlagName, 8),
	)
	flags2 := tinyflags.New(
		tinyflags.NewMemoryStore(redisClient),
		tinyflags.NewRedisStore(redisClient, "example"),
		tinyflags.NewPostgresStore(postgresClient),
		tinyflags.NewConstantStore().With(rateLimitFlagName, 8),
	)
	// read the flag from both managers
	rateLimitFlag1 := tinyflags.NewIntFlag(rateLimitFlagName)
	rateLimitFlag2 := tinyflags.NewIntFlag(rateLimitFlagName)
	if err := flags1.Read(ctx, &rateLimitFlag1); err != nil {
		fmt.Printf("error reading flags: %v\n", err)
		return
	}
	if err := flags2.Read(ctx, &rateLimitFlag2); err != nil {
		fmt.Printf("error reading flags: %v\n", err)
		return
	}
	fmt.Printf("rate limit 1: %d\n", rateLimitFlag1.Get())
	fmt.Printf("rate limit 2: %d\n", rateLimitFlag2.Get())
	time.Sleep(longWait)
	// write a new value to manager 1
	rateLimitFlag1 = tinyflags.NewIntFlag(rateLimitFlagName).With(16)
	if err := flags1.Write(ctx, &rateLimitFlag1); err != nil {
		fmt.Printf("error writing flags: %v\n", err)
		return
	}
	time.Sleep(tinyWait)
	// read the flag again from both managers
	rateLimitFlag1 = tinyflags.NewIntFlag(rateLimitFlagName)
	rateLimitFlag2 = tinyflags.NewIntFlag(rateLimitFlagName)
	if err := flags1.Read(ctx, &rateLimitFlag1); err != nil {
		fmt.Printf("error reading flags: %v\n", err)
		return
	}
	if err := flags2.Read(ctx, &rateLimitFlag2); err != nil {
		fmt.Printf("error reading flags: %v\n", err)
		return
	}
	fmt.Printf("rate limit 1: %d\n", rateLimitFlag1.Get())
	fmt.Printf("rate limit 2: %d\n", rateLimitFlag2.Get())
}
