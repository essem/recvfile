package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	port := flag.Int("p", 3000, "Listening port")
	bufSize := flag.Int("b", 300, "Buffer size")
	outFile := flag.String("o", "", "Output file name")
	echo := flag.Bool("e", false, "Send back recevied bytes")
	verbose := flag.Bool("v", false, "Verbose")
	multiple := flag.Bool("m", false, "Multiple connection")
	flag.Parse()

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

	fmt.Printf("Connection from %s\n", conn.RemoteAddr())

	totalReceived := 0

	for {
		received := recv(conn, *bufSize, *verbose, *echo, file)
		if received == 0 {
			if *multiple {
				conn.Close()
				conn, err = ln.Accept()
				if err != nil {
					fmt.Printf("Failed to accept: %v\n", err)
					return
				}
			} else {
				fmt.Println("Connection closed")
				break
			}
		}
		totalReceived += received
	}

	if file != nil {
		file.Close()
	}

	conn.Close()

	fmt.Printf("Total recv: %d\n", totalReceived)
}

func recv(conn net.Conn, bufSize int, verbose bool, echo bool, file *os.File) int {

	buf := make([]byte, bufSize)
	received, err := conn.Read(buf)
	if err == io.EOF {
		if received != 0 {
			fmt.Printf("Unexpected: %d", received)
			os.Exit(1)
		}
		return 0
	} else if err != nil {
		fmt.Printf("Failed to recv: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Recevied: %d\n", received)
	}

	if echo {
		sent := 0
		for sent < received {
			n, err := conn.Write(buf[sent:received])
			if err != nil {
				fmt.Printf("Failed to send: %v\n", err)
				os.Exit(1)
			}

			if verbose {
				fmt.Printf("Sent: %d\n", n)
			}

			sent += n
		}
	}

	if file != nil {
		written := 0
		for written < received {
			n, err := file.Write(buf[written:received])
			if err != nil {
				fmt.Printf("Failed to write: %v\n", err)
				os.Exit(1)
			}

			if verbose {
				fmt.Printf("Written: %d\n", n)
			}

			written += n
		}
	}

	return received
}
