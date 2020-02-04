package numbers

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func NewNumbersProtocol(bufferSize int) (TCPProtocol, chan int, chan bool) {
	terminate := make(chan bool)
	numbers := make(chan int, bufferSize)
	return func(ctx context.Context, c net.Conn) error {
		if checkIfTerminated(ctx) {
			return fmt.Errorf("connection: %s, terminated", c.RemoteAddr().String())
		}

		reader := bufio.NewReader(c)
		data, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		if checkIfTerminated(ctx) {
			return fmt.Errorf("connection: %s, terminated", c.RemoteAddr().String())
		}

		data = strings.TrimSuffix(data, "\n")
		if data == "terminate" {
			terminate <- true
			close(terminate)
			close(numbers)
			return fmt.Errorf("connection: %s, terminated", c.RemoteAddr().String())
		}

		if len(data) != 9 {
			return fmt.Errorf("client: %s, no 9 char length string %s", c.RemoteAddr().String(), data)
		}

		number, err := strconv.Atoi(data)
		if err != nil {
			log.Println(err)
			return err
		}

		log.Printf("connection: %s, number: %d", c.RemoteAddr().String(), number)

		if checkIfTerminated(ctx) {
			return fmt.Errorf("client: %s, terminated")
		}

		numbers <- number

		return nil
	}, numbers, terminate
}

type NumberStore func(ctx context.Context)

func StartStore(ctx context.Context, numberStore NumberStore) {
	go numberStore(ctx)
}

func NewNumberStore(reportPeriod int, in chan int) NumberStore {
	numbers := make(map[int]bool)
	currentUnique := 0
	currentDuplicated := 0
	ticker := time.NewTicker(time.Duration(reportPeriod) * time.Second)
	return func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				break
			case number, more := <-in:
				if more {
					if _, ok := numbers[number]; ok {
						currentDuplicated++
					} else {
						currentUnique++
						numbers[number] = true
					}
				} else {
					break
				}
			case tick := <-ticker.C:
				log.Print("Report of ", tick)
				log.Printf("Received %d unique numbers, %d duplicates. Unique total: %d",
					currentUnique, currentDuplicated, len(numbers))
			}
		}
	}
}


