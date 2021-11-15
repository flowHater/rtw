package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Config struct {
	RootPath string    `json:"rootPath"`
	Services []Service `json:"services"`
}

type Service struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Path    string   `json:"path"`
	Args    []string `json:"args"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt)
	// go func() {
	// 	oscall := <-c
	// 	log.Printf("system call:%+v", oscall)
	// 	cancel()
	// }()

	fileName := flag.String("file", "", "path to the json file containing list of services")
	stopOnError := flag.Bool("stopOnError", false, "Should stop on error")
	flag.Parse()

	if fileName == nil || *fileName == "" {
		log.Fatal("not services.json provided")
	}

	config := &Config{}

	file, err := ioutil.ReadFile(*fileName)
	if err != nil {
		log.Fatalf("cannot open(%s): %s", *fileName, err)
	}

	err = json.Unmarshal(file, &config)
	if err != nil {
		log.Fatalf("cannot json.Unmarshal(%s): %s", *fileName, err)
	}

	rs := runServices(ctx, config)
	out, outErr := readManyReadClosers(rs)

	go func() {
		if *stopOnError {
			err := <-outErr
			cancel()
			log.Fatal(errors.New(string(err)))
		} else {
			for err := range outErr {
				log.Println(string(err))
			}
		}
	}()

	for line := range out {
		if len(line) > 0 {
			fmt.Println(string(line))
		}
	}
}

func runServices(ctx context.Context, c *Config) []Logger {
	readers := []Logger{}

	for _, service := range c.Services {
		p := filepath.Join(c.RootPath, service.Path)
		for i := range service.Args {
			service.Args[i] = strings.ReplaceAll(service.Args[i], "$path", p)
			service.Args[i] = os.ExpandEnv(service.Args[i])
		}

		cmd := exec.CommandContext(ctx, service.Command, service.Args...)
		cmd.Dir = p
		fmt.Printf("Running in %q: %s %v\n", p, service.Command, service.Args)

		r, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatalf("cannot StdoutPipe(%s) with %s: %s", service.Command, service.Args, err)
		}

		rErr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatalf("cannot StderrPipe(%s) with %s: %s", service.Command, service.Args, err)
		}

		go func() {
			err := cmd.Start()
			if err != nil {
				log.Fatalf("cannot Start(%s) with %s: %s", service.Command, service.Args, err)
			}

		}()

		readers = append(readers, Logger{
			r:      r,
			rErr:   rErr,
			prefix: service.Name,
		})
	}

	return readers
}

type Logger struct {
	prefix string
	r      io.ReadCloser
	rErr   io.ReadCloser
}

func readManyReadClosers(loggers []Logger) (chan []byte, chan []byte) {
	out := make(chan []byte)
	errCh := make(chan []byte)
	for _, l := range loggers {
		go func(logger Logger) {
			buf := bufio.NewScanner(logger.r)

			for buf.Scan() {
				line := bytes.Buffer{}
				line.WriteString(logger.prefix)
				line.WriteString(": ")
				line.Write(sanitiseLine(buf.Bytes()))
				if line.Len() != len(logger.prefix)+2 {
					out <- line.Bytes()
				}
			}
		}(l)

		go func(logger Logger) {
			buf := bufio.NewScanner(logger.rErr)
			prefix := logger.prefix + ":error"

			for buf.Scan() {
				line := bytes.Buffer{}
				line.WriteString(prefix)
				line.WriteString(": ")
				line.Write(buf.Bytes())
				if line.Len() != len(prefix)+2 {
					errCh <- line.Bytes()
				}
			}
		}(l)
	}

	return out, errCh
}

var bytesToRemove = [][]byte{
	[]byte("\033c"),
	[]byte("\b"),
	[]byte("\r"),
}

func sanitiseLine(l []byte) []byte {
	for _, r := range bytesToRemove {
		l = bytes.ReplaceAll(l, r, nil)
	}

	return l
}
