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
	flag.Parse()

	bindAddr := fmt.Sprintf(":%d", *port)
	fmt.Printf("Listen on %s\n", bindAddr)

	writeFile := false
	var file os.File
	if *outFile != "" {
		file, err := os.Create(*outFile)
		if err != nil {
			fmt.Printf("Failed to create file: %v\n", err)
		}

		defer file.Close()

		writeFile = true

		fmt.Printf("Write file to %s\n", *outFile)
	}

	if *echo {
		fmt.Printf("Turn on echo\n")
	}

	ln, err := net.Listen("tcp", bindAddr)
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
	}

	defer ln.Close()

	conn, err := ln.Accept()
	if err != nil {
		fmt.Printf("Failed to accept: %v\n", err)
	}

	defer conn.Close()

	fmt.Printf("Connection from %s\n", conn.RemoteAddr())

	totalReceived := 0

	for {
		buf := make([]byte, *bufSize)
		received, err := conn.Read(buf)
		if err == io.EOF {
			fmt.Println("Connection closed")
			if received != 0 {
				fmt.Printf("Unexpected: %d", received)
			}
			break
		} else if err != nil {
			fmt.Printf("Failed to recv: %v\n", err)
		}

		totalReceived += received

		if *verbose {
			fmt.Printf("Recevied: %d\n", received)
		}

		if *echo {
			sent := 0
			for sent < received {
				n, err := conn.Write(buf[sent:received])
				if err != nil {
					fmt.Printf("Failed to send: %v\n", err)
				}

				if *verbose {
					fmt.Printf("Sent: %d\n", n)
				}

				sent += n
			}
		}

		if writeFile {
			written := 0
			for written < received {
				n, err := file.Write(buf[written:received])
				if err != nil {
					fmt.Printf("Failed to write: %v\n", err)
				}

				if *verbose {
					fmt.Printf("Written: %d\n", n)
				}

				written += n
			}
		}
	}

	fmt.Printf("Total recv: %d\n", totalReceived)
}
