package numbers

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const numberControllersBufferSize = 10000
const numberWriterBufferSize = 10000
const reportPeriod = 10
const connectionReadTimeout = time.Duration(5) * time.Second
const numberLogFileName = "numbers.log"

func StartNumberServer(concurrentConnections int, address string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numbersController, numbersOut, terminateOut := NewNumbersController(numberControllersBufferSize)
	deDuplicatedNumbers := NewNumberStore(ctx, reportPeriod, numbersOut, numberWriterBufferSize)
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	NewFileWriter(ctx, deDuplicatedNumbers, dir+"/"+numberLogFileName)

	cnnListener := NewSingleConnectionListener(numbersController, connectionReadTimeout)
	multipleListener, err := NewMultipleConnectionListener(concurrentConnections, cnnListener)
	if err != nil {
		log.Println(err)
	}

	go func() {
		for {
			select {
			case <-terminateOut:
				cancel()
				return
			}
		}
	}()

	StartConnectionListener(ctx, multipleListener, address)
}

func NewNumbersController(bufferSize int) (TCPController, chan int, chan bool) {
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
			return fmt.Errorf("client: %s, terminated", c.RemoteAddr().String())
		}

		numbers <- number

		return nil
	}, numbers, terminate
}

func NewNumberStore(ctx context.Context, reportPeriod int, in chan int, outBufferSize int) chan int {
	out := make(chan int, outBufferSize)
	numbers := make(map[int]bool)
	currentUnique := 0
	currentDuplicated := 0
	ticker := time.NewTicker(time.Duration(reportPeriod) * time.Second)
	go func(ctx context.Context) {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case number, more := <-in:
				if more {
					if _, ok := numbers[number]; ok {
						currentDuplicated++
					} else {
						currentUnique++
						numbers[number] = true
						out <- number
					}
				} else {
					return
				}
			case tick := <-ticker.C:
				log.Printf("Report %v Received %d unique numbers, %d duplicates. Unique total: %d",
					tick, currentUnique, currentDuplicated, len(numbers))
			}
		}
	}(ctx)
	return out
}

func NewFileWriter(ctx context.Context, in chan int, filePath string) {
	f, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	go func(ctx context.Context) {
		defer closeFile(f)
		for {
			select {
			case <-ctx.Done():
				return
			case number, more := <-in:
				if more {
					_, err := fmt.Fprintf(f, "%09d\n", number)
					if err != nil {
						log.Println(err)
						return
					}
				} else {
					return
				}
			}
		}
	}(ctx)
}

func closeFile(f *os.File) {
	if err := f.Close(); err != nil {
		log.Println(err)
	}
}
