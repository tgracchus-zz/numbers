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
	"sync"
	"time"
)

const reportPeriod = 10
const numberLogFileName = "numbers.log"

func StartNumberServer(ctx context.Context, concurrentConnections int, address string) {
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if concurrentConnections < 0 {
		log.Panicf("concurrency level should be more than 0, not %d", concurrentConnections)
	}

	terminate := cancelContextWhenTerminateSignal(ctx, cancel)
	controllers := make([]TCPController, concurrentConnections)
	listeners := make([]ConnectionListener, concurrentConnections)
	numbersOuts := make([]chan int, concurrentConnections)
	for i := 0; i < concurrentConnections; i++ {
		numbersController, numbersOut := NewNumbersController(terminate)
		cnnListener := NewSingleConnectionListener(numbersController)
		controllers[i] = numbersController
		listeners[i] = cnnListener
		numbersOuts[i] = numbersOut
	}

	deDuplicatedNumbers := NewNumberStore(cctx, reportPeriod, numbersOuts)
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	NewFileWriter(cctx, deDuplicatedNumbers, dir+"/"+numberLogFileName)

	multipleListener, err := NewMultipleConnectionListener(listeners)
	if err != nil {
		log.Printf("%v", err)
	}

	StartServer(cctx, multipleListener, address)
}

func cancelContextWhenTerminateSignal(ctx context.Context, cancel context.CancelFunc) chan int {
	terminate := make(chan int)
	go func() {
		defer close(terminate)
		for {
			select {
			case <-terminate:
				cancel()
			case <-ctx.Done():
				return
			}
		}
	}()
	return terminate
}

func NewNumbersController(terminate chan int) (TCPController, chan int) {
	numbers := make(chan int)
	duration := time.Duration(60) * time.Second
	return func(ctx context.Context, c net.Conn) error {
		reader := bufio.NewReader(c)
		for {
			if checkIfTerminated(ctx) {
				return nil
			}
			data, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return errors.Wrap(err, "ReadString")
			}

			err = c.SetReadDeadline(time.Now().Add(duration))
			if err != nil {
				return errors.Wrap(err, "SetReadDeadline")
			}

			data = strings.TrimSuffix(data, "\n")

			if len(data) != 9 {
				log.Printf("%v", errors.Wrap(fmt.Errorf("client: %s, no 9 char length string %s", c.RemoteAddr().String(), data), "check for 9 digits"))
				return nil
			}

			if data == "terminate" {
				terminate <- 1
				close(numbers)
				return errors.Wrap(fmt.Errorf("connection: %s, terminated", c.RemoteAddr().String()), "terminated")
			}

			number, err := strconv.Atoi(data)
			if err != nil {
				log.Printf("%v", errors.Wrap(err, "strconv.Atoi"))
				return nil
			}

			numbers <- number
		}

	}, numbers
}

func NewNumberStore(ctx context.Context, reportPeriod int, ins []chan int) chan int {
	out := make(chan int)
	in := fanIn(ctx, ins)

	numbers := make(map[int]bool)
	var total int64 = 0
	var currentUnique int64 = 0
	var currentDuplicated int64 = 0
	ticker := time.NewTicker(time.Duration(reportPeriod) * time.Second)
	store := func(ctx context.Context) {
		defer ticker.Stop()
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

func fanIn(ctx context.Context, ins []chan int) chan int {
	var wg sync.WaitGroup
	wg.Add(len(ins))
	out := make(chan int)
	go func() {
		for _, ch := range ins {
			go func(in chan int) {
				defer wg.Done()
				for {
					select {
					case element, more := <-in:
						if more {
							out <- element
						} else {
							return
						}
					case <-ctx.Done():
						return
					}
				}
			}(ch)
		}
		wg.Wait()
		close(out)
	}()
	return out
}

func NewFileWriter(ctx context.Context, in chan int, filePath string) {
	f, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	b := bufio.NewWriter(f)
	ticker := time.NewTicker(time.Duration(reportPeriod) * time.Second)
	go func(ctx context.Context) {
		defer ticker.Stop()
		defer closeFile(b, f)
		for {
			select {
			case <-ctx.Done():
				return
			case number, more := <-in:
				if more {
					_, err := fmt.Fprintf(b, "%09d\n", number)
					if err != nil {
						log.Printf("%v", errors.Wrap(err, "Fprintf"))
					}
				} else {
					return
				}
			case <-ticker.C:
				if err := b.Flush(); err != nil {
					log.Printf("%v", err)
				}
			}

		}
	}(ctx)
}

func closeFile(b *bufio.Writer, f *os.File) {
	if err := b.Flush(); err != nil {
		log.Printf("%v", err)
	}
	if err := f.Close(); err != nil {
		log.Printf("%v", err)
	}
}
