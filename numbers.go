package numbers

import (
	"bufio"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const numberControllersBufferSize = 10000
const numberWriterBufferSize = 100000
const reportPeriod = 10
const numberLogFileName = "numbers.log"

func StartNumberServer(ctx context.Context, concurrentConnections int, address string) {
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	numbersController, numbersOut, terminateOut := NewNumbersController(numberControllersBufferSize)
	deDuplicatedNumbers := NewNumberStore(cctx, reportPeriod, numbersOut, numberWriterBufferSize)
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	NewFileWriter(cctx, deDuplicatedNumbers, dir+"/"+numberLogFileName)

	cnnListener := NewSingleConnectionListener(numbersController)
	multipleListener, err := NewMultipleConnectionListener(concurrentConnections, cnnListener)
	if err != nil {
		log.Printf("%+v", err)
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

	StartServer(cctx, multipleListener, address)
}

func NewNumbersController(bufferSize int) (TCPController, chan int, chan bool) {
	terminate := make(chan bool)
	numbers := make(chan int, bufferSize)
	//duration := time.Duration(5) * time.Second
	return func(ctx context.Context, c net.Conn) error {
		defer closeConnection(c)
		reader := bufio.NewReader(c)
		/*err := c.SetReadDeadline(time.Now().Add(duration))
		if err != nil {
			return errors.Wrap(err, "SetReadDeadline")
		}*/
		for {
			if checkIfTerminated(ctx) {
				return errors.Wrapf(ctx.Err(), "client: %s, terminated", c.RemoteAddr().String())
			}
			data, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return errors.Wrap(err, "ReadString")
			}

			data = strings.TrimSuffix(data, "\n")

			if len(data) != 9 {
				log.Printf("%v", errors.Wrap(fmt.Errorf("client: %s, no 9 char length string %s", c.RemoteAddr().String(), data), "check for 9 digits"))
				return nil
			}

			if checkIfTerminated(ctx) {
				return errors.Wrapf(ctx.Err(), "connection: %s, terminated", c.RemoteAddr().String())
			}

			if data == "terminate" {
				terminate <- true
				close(terminate)
				close(numbers)
				return errors.Wrap(fmt.Errorf("connection: %s, terminated", c.RemoteAddr().String()), "terminated")
			}

			number, err := strconv.Atoi(data)
			if err != nil {
				log.Printf("%v", errors.Wrap(err, "strconv.Atoi"))
				return nil
			}

			if checkIfTerminated(ctx) {
				return errors.Wrapf(ctx.Err(), "client: %s, terminated", c.RemoteAddr().String())
			}

			numbers <- number
		}

	}, numbers, terminate
}

func NewNumberStore(ctx context.Context, reportPeriod int, in chan int, outBufferSize int) chan int {
	out := make(chan int, outBufferSize)
	numbers := make(map[int]bool)
	var total int64 = 0
	var currentUnique int64 = 0
	var currentDuplicated int64 = 0
	ticker := time.NewTicker(time.Duration(reportPeriod) * time.Second)
	store := func(ctx context.Context) {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case number, more := <-in:
				if more {
					total++
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
				log.Printf("Report %v Received %d unique numbers, %d duplicates. Unique total: %d. Total: %d",
					tick, currentUnique, currentDuplicated, len(numbers), total)
				currentUnique = 0
				currentDuplicated = 0
			}
		}
	}
	go store(ctx)
	return out
}

func NewFileWriter(ctx context.Context, in chan int, filePath string) {
	f, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	b := bufio.NewWriter(f)
	writer := func(ctx context.Context) {
		defer closeFile(f)
		for {
			select {
			case <-ctx.Done():
				return
			case number, more := <-in:
				if more {
					_, err := fmt.Fprintf(b, "%09d\n", number)
					if err != nil {
						log.Printf("%+v", errors.Wrap(err, "Fprintf"))
					}
				} else {
					return
				}
			}
		}
	}

	go writer(ctx)
}

func closeFile(f *os.File) {
	if err := f.Close(); err != nil {
		log.Printf("%+v", err)
	}
}
