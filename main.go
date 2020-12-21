package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/robfig/cron/v3"
)

func main() {
	fmt.Println("Starting Depper...")

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	c := cron.New()
	c.AddFunc("* * * * *", func() { fmt.Println("Hello world from Depper") })
	c.Start()

	sig := <-s
	signal.Stop(s)
	fmt.Printf("\nCaught signal: %s.\n", sig)
}
