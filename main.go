package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

type condition struct {
	bufSize int
	verbose bool
}

type receiveResult struct {
	total int
}

type sendResult struct {
	total int
}

type writeResult struct {
	total int
}

func main() {
	port := flag.Int("p", 3000, "Listening port")
	bufSize := flag.Int("b", 300, "Buffer size")
	chanSize := flag.Int("c", 1, "Channel size")
	outFile := flag.String("o", "", "Output file name")
	echo := flag.Bool("e", false, "Send back recevied bytes")
	verbose := flag.Bool("v", false, "Verbose")
	flag.Parse()

	cond := condition{
		bufSize: *bufSize,
		verbose: *verbose,
	}

	bindAddr := fmt.Sprintf(":%d", *port)
	fmt.Printf("Listen on %s\n", bindAddr)

	var file *os.File
	if *outFile != "" {
		f, err := os.Create(*outFile)
		if err != nil {
			fmt.Printf("Failed to create file: %v\n", err)
			return
		}

		file = f

		fmt.Printf("Write file to %s\n", *outFile)
	}

	if *echo {
		fmt.Printf("Turn on echo\n")
	}

	// Bind and Listen

	ln, err := net.Listen("tcp", bindAddr)
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		return
	}

	defer ln.Close()

	conn, err := ln.Accept()
	if err != nil {
		fmt.Printf("Failed to accept: %v\n", err)
		return
	}

	defer conn.Close()

	fmt.Printf("Connection from %s\n", conn.RemoteAddr())

	// Prepare channels and launch goroutines

	receiveResultCh := make(chan receiveResult, *chanSize)
	sendResultCh := make(chan sendResult, *chanSize)
	writeResultCh := make(chan writeResult, *chanSize)

	var nextInCh chan []byte

	{
		var receiveOutCh chan []byte
		if *echo || file != nil {
			receiveOutCh = make(chan []byte)
		}

		go receiver(cond, conn, receiveOutCh, receiveResultCh)
		nextInCh = receiveOutCh
	}

	if *echo {
		var sendOutCh chan []byte
		if file != nil {
			sendOutCh = make(chan []byte)
		}

		go sender(cond, conn, nextInCh, sendOutCh, sendResultCh)
		nextInCh = sendOutCh
	}

	if file != nil {
		go writer(cond, file, nextInCh, writeResultCh)
	}

	// Wait goroutines and check result

	receiveResult := <-receiveResultCh
	if *echo {
		sendResult := <-sendResultCh
		if receiveResult.total != sendResult.total {
			fmt.Printf("Result not match: receive %d send %d\n", receiveResult.total, sendResult.total)
			os.Exit(1)
		}
	}
	if file != nil {
		writeResult := <-writeResultCh
		if receiveResult.total != writeResult.total {
			fmt.Printf("Result not match: receive %d write %d\n", receiveResult.total, writeResult.total)
			os.Exit(1)
		}
		file.Close()
	}

	fmt.Printf("Total recv: %d\n", receiveResult.total)
}

func receiver(cond condition, conn net.Conn, outCh chan []byte, resultCh chan receiveResult) {
	var result receiveResult

	for {
		buf := make([]byte, cond.bufSize)
		received, err := conn.Read(buf)
		if err == io.EOF {
			fmt.Println("Connection closed")
			if received != 0 {
				fmt.Printf("Unexpected: %d", received)
				os.Exit(1)
			}
			break
		} else if err != nil {
			fmt.Printf("Failed to recv: %v\n", err)
			os.Exit(1)
		}

		if outCh != nil {
			outCh <- buf[:received]
		}

		result.total += received

		if cond.verbose {
			fmt.Printf("Recevied: %d\n", received)
		}
	}

	if outCh != nil {
		close(outCh)
	}

	resultCh <- result
}

func sender(cond condition, conn net.Conn, inCh chan []byte, outCh chan []byte, resultCh chan sendResult) {
	var result sendResult

	for buf := range inCh {
		sent := 0
		for sent < len(buf) {
			n, err := conn.Write(buf[sent:])
			if err != nil {
				fmt.Printf("Failed to send: %v\n", err)
				os.Exit(1)
			}

			if cond.verbose {
				fmt.Printf("Sent: %d\n", n)
			}

			sent += n
		}

		if outCh != nil {
			outCh <- buf[:sent]
		}

		result.total += sent
	}

	if outCh != nil {
		close(outCh)
	}

	resultCh <- result
}

func writer(cond condition, file *os.File, inCh chan []byte, resultCh chan writeResult) {
	var result writeResult

	for buf := range inCh {
		written := 0
		for written < len(buf) {
			n, err := file.Write(buf[written:])
			if err != nil {
				fmt.Printf("Failed to write: %v\n", err)
				os.Exit(1)
			}

			if cond.verbose {
				fmt.Printf("Written: %d\n", n)
			}

			written += n
		}

		result.total += written
	}

	resultCh <- result
}
