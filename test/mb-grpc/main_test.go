package mbgrpc

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	var mbArgs []string
	mbArgs = append(mbArgs, "start")
	mbArgs = append(mbArgs, "--port", strconv.Itoa(2525))
	mbArgs = append(mbArgs, "--protofile", "protocols.json")
	mbArgs = append(mbArgs, "--configfile", "imposters.json")
	cmd := exec.CommandContext(ctx, "mb", mbArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Println(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Println("scanner error:", err)
		}
	}()
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Println(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Println("scanner error:", err)
		}
	}()

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(2 * time.Second)
	res := m.Run()

	err = cmd.Process.Signal(os.Interrupt)
	if err != nil {
		log.Fatal(err)
	}
	_, err = cmd.Process.Wait()
	if err != nil {
		log.Fatal(err)
	}
	wg.Wait()

	os.Exit(res)
}
