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

// StartNumberServer start the number server tcp application with
// number of concurrent server connections and at the given address.
func StartNumberServer(concurrentConnections int, address string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if concurrentConnections < 0 {
		log.Panicf("concurrency level should be more than 0, not %d", concurrentConnections)
	}
	terminate := make(chan int)
	controllers := make([]TCPController, concurrentConnections)
	listeners := make([]ConnectionListener, concurrentConnections)
	numbersOuts := make([]chan int, concurrentConnections)
	for i := 0; i < concurrentConnections; i++ {
		numbersController, numbers := NewNumbersController(terminate)
		cnnListener := NewSingleConnectionListener(numbersController)
		controllers[i] = numbersController
		listeners[i] = cnnListener
		numbersOuts[i] = numbers
	}
	deDuplicatedNumbers := NewNumberStore(reportPeriod, numbersOuts)
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	done := NewFileWriter(deDuplicatedNumbers, dir+"/"+numberLogFileName)
	multipleListener := NewMultipleConnectionListener(listeners)

	cancelContextWhenTerminateSignal(cancel, multipleListener, terminate, done)
	StartServer(ctx, multipleListener, address)
}

func cancelContextWhenTerminateSignal(cancel context.CancelFunc, listener ConnectionListener,
	terminate chan int, done chan int) chan int {
	go func() {
		defer close(terminate)
		for {
			select {
			case <-terminate:
				listener.Close()
			case <-done:
				cancel()
				return
			}

		}
	}()
	return terminate
}

// NewNumbersController is the numbers app controller. It handles the parsing protocol defined in the requirements.
// Accepts a channel terminate to send termination signal and return a channel from where the numbers will be issued
// once they are parsed.
func NewNumbersController(terminate chan int) (TCPController, chan int) {
	numbers := make(chan int)
	duration := time.Duration(30) * time.Second
	return &numbersController{terminate: terminate, numbers: numbers, duration: duration}, numbers
}

type numbersController struct {
	terminate chan int
	numbers   chan int
	duration  time.Duration
}

func (nc *numbersController) Close() {
	close(nc.numbers)
}

func (nc *numbersController) Handle(ctx context.Context, c net.Conn) error {
	reader := bufio.NewReader(c)
	for {
		if checkIfTerminated(ctx) {
			return errors.Wrap(fmt.Errorf("connection: %s, terminated", c.RemoteAddr().String()), "terminate check")
		}
		err := c.SetReadDeadline(time.Now().Add(nc.duration))
		if err != nil {
			return errors.Wrap(err, "SetReadDeadline")
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
			return errors.Wrap(fmt.Errorf("client: %s, no 9 char length string %s", c.RemoteAddr().String(), data), "check for 9 digits")
		}
		if data == "terminate" {
			nc.terminate <- 1
			return errors.Wrap(fmt.Errorf("connection: %s, terminated", c.RemoteAddr().String()), "terminate signal")
		}
		number, err := strconv.Atoi(data)
		if err != nil {
			return errors.Wrap(err, "strconv.Atoi")
		}
		nc.numbers <- number
	}
}

// NewNumberStore given a list of channels it listens to all of them and deduplicated the numbers received.
// If the number is not duplicated handles it to the returned channel for further processing.
// It also keeps track of total and unique numbers and the current 10s windows of unique and duplicated numbers.
func NewNumberStore(reportPeriod int, ins []chan int) chan int {
	out := make(chan int)
	in := fanIn(ins)
	numbers := make(map[int]bool)
	var total int64 = 0
	var currentUnique int64 = 0
	var currentDuplicated int64 = 0
	ticker := time.NewTicker(time.Duration(reportPeriod) * time.Second)
	go func() {
		defer ticker.Stop()
		defer close(out)
		for {
			select {
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
	}()
	return out
}

func fanIn(ins []chan int) chan int {
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
					}
				}
			}(ch)
		}
		wg.Wait()
		close(out)
	}()
	return out
}

// NewFileWriter writes al the numbers received at in channel and writes them to filePath.
// Returns a done channel when it is terminated
func NewFileWriter(in chan int, filePath string) chan int {
	done := make(chan int)
	f, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	b := bufio.NewWriter(f)
	ticker := time.NewTicker(time.Duration(reportPeriod) * time.Second)
	go func() {
		defer ticker.Stop()
		defer closeFile(b, f)
		for {
			select {
			case number, more := <-in:
				if more {
					_, err := fmt.Fprintf(b, "%09d\n", number)
					if err != nil {
						log.Printf("%v", errors.Wrap(err, "Fprintf"))
					}
				} else {
					done <- 1
					return
				}
			case <-ticker.C:
				if err := b.Flush(); err != nil {
					log.Printf("%v", err)
				}
			}

		}
	}()
	return done
}

func closeFile(b *bufio.Writer, f *os.File) {
	if err := b.Flush(); err != nil {
		log.Printf("%v", err)
	}
	if err := f.Close(); err != nil {
		log.Printf("%v", err)
	}
}

func checkIfTerminated(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
