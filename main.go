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
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
	c := cron.New()
	if _, err := c.AddFunc("* * * * *", func() { fmt.Println("Hello world from Depper") }); err != nil {
		_ = fmt.Errorf("Error: %v", err)
		os.Exit(1)
	}
	c.Start()

	sig := <-s
	signal.Stop(s)
	fmt.Printf("Caught signal: %s.\n", sig)
}
