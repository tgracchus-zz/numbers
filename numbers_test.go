package numbers_test

import (
	"bufio"
	"context"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"testing"
	"tgracchus/numbers"
	"time"
)

func TestNumbersControllerReadNumber(t *testing.T) {
	server, client := net.Pipe()
	numbersProtocol, numbersIn, _ := numbers.NewNumbersController(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wireNumber := "098765432"
	expectedNumber, err := strconv.Atoi(wireNumber)
	if err != nil {
		t.Fatal(err)
	}

	sendData(t, client, wireNumber)
	err = numbersProtocol(ctx, server)
	if err != nil {
		t.Fatal(err)
	}
	number := <-numbersIn
	if expectedNumber != number {
		t.Fatal(fmt.Errorf("number should be: %d not %d", expectedNumber, number))
	}
}

func TestNumbersControllerNotNumber(t *testing.T) {
	server, client := net.Pipe()
	numbersProtocol, _, _ := numbers.NewNumbersController(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wireText := "IAMTEXT!!"
	sendData(t, client, wireText)
	err := numbersProtocol(ctx, server)
	if err == nil {
		t.Fatal("An *strconv.NumError was expected when parsing a text")
	}
	_, ok := err.(*strconv.NumError)
	if !ok {
		t.Fatal("An *strconv.NumError was expected when parsing a text")
	}
}

func TestNumbersControllerClosedConnection(t *testing.T) {
	server, _ := net.Pipe()
	numbersProtocol, _, _ := numbers.NewNumbersController(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := server.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = numbersProtocol(ctx, server)
	if err == nil {
		t.Fatal("io: read/write on closed pipe was expected when parsing a text")
	}
}

func TestNumbersControllerContextCancelled(t *testing.T) {
	server, client := net.Pipe()
	numbersProtocol, _, _ := numbers.NewNumbersController(10)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	wireNumber := "098765432"

	sendData(t, client, wireNumber)
	err := numbersProtocol(ctx, server)
	if err == nil {
		t.Fatal("connection: pipe, terminated is expected")
	}
}

func TestNumbersControllerTerminate(t *testing.T) {
	server, client := net.Pipe()
	numbersProtocol, _, terminated := numbers.NewNumbersController(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sendData(t, client, "terminate")

	termkilled := false
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		select {
		case termkilled = <-terminated:
			wg.Done()
		case <-time.After(1 * time.Second):
			wg.Done()
		}
	}()
	err := numbersProtocol(ctx, server)
	if err == nil {
		t.Fatal("connection: pipe, terminated is expected")
	}
	wg.Wait()
	if !termkilled {
		t.Fatal("a termkill signal is expected in the terminated channel")
	}
}

func sendData(t *testing.T, client net.Conn, wireNumber string) {
	go func() {
		_, err := client.Write([]byte(wireNumber + "\n"))
		if err != nil {
			t.Fatal(err)
		}
	}()
}

func TestNewNumberStoreNumber(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numbersIn := make(chan int, 2)
	defer close(numbersIn)

	expectedNumber := 123456789
	numbersIn <- expectedNumber
	numberOut := numbers.NewNumberStore(ctx, 10, numbersIn, 2)

	expectNumber(numberOut, expectedNumber, t)
}

func TestNewNumberStoreTwoNumbers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numbersIn := make(chan int, 2)
	defer close(numbersIn)
	expectedNumber1 := 123456789
	expectedNumber2 := 987654321
	numbersIn <- expectedNumber1
	numbersIn <- expectedNumber2

	numberOut := numbers.NewNumberStore(ctx, 10, numbersIn, 2)

	expectNumber(numberOut, expectedNumber1, t)
	expectNumber(numberOut, expectedNumber2, t)
}

func TestNewNumberStoreDeduplicated(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numbersIn := make(chan int, 2)
	defer close(numbersIn)
	expectedNumber := 123456789

	numbersIn <- expectedNumber
	numbersIn <- expectedNumber

	numberOut := numbers.NewNumberStore(ctx, 10, numbersIn, 2)

	expectNumber(numberOut, expectedNumber, t)
	numberNotExpected(numberOut, t)
}

func TestNewNumberStoreCloseInChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numbersIn := make(chan int, 2)
	close(numbersIn)

	numberOut := numbers.NewNumberStore(ctx, 10, numbersIn, 2)
	numberNotExpected(numberOut, t)
}

func numberNotExpected(numberOut chan int, t *testing.T) {
	select {
	case _, open := <-numberOut:
		if open {
			t.Fatal("number not expected here")
		}
	case <-time.After(1 * time.Second):
	}
}
func expectNumber(numberOut chan int, expectedNumber int, t *testing.T) {
	select {
	case number := <-numberOut:
		if number != expectedNumber {
			t.Fatal(fmt.Errorf("number should be: %d not %d", expectedNumber, number))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout while waiting for a response in numberOut")
	}
}

func TestNewFileWriter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numbersIn := make(chan int, 2)
	defer close(numbersIn)

	expectedNumber := 123456789
	numbersIn <- expectedNumber

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	filePath := testFilePath(dir)
	defer cleanUpFile(filePath)
	numbers.NewFileWriter(ctx, numbersIn, filePath)

	expectedFileContent(t, err, filePath, expectedNumber)
}

func TestNewFileWriterContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	numbersIn := make(chan int, 2)
	defer close(numbersIn)
	numbersIn <- 123456789

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	filePath := testFilePath(dir)
	defer cleanUpFile(filePath)
	numbers.NewFileWriter(ctx, numbersIn, filePath)

	fileEmpty(t, err, filePath)
}

func TestNewFileWriterChannelClosed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numbersIn := make(chan int, 2)
	close(numbersIn)

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	filePath := testFilePath(dir)
	defer cleanUpFile(filePath)
	numbers.NewFileWriter(ctx, numbersIn, filePath)
	fileEmpty(t, err, filePath)
}

func TestNewFileWriterOverwriteWhenStarted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numbersIn := make(chan int, 2)
	defer close(numbersIn)

	nonExpectedNumber := 123456789

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	filePath := testFilePath(dir)
	defer cleanUpFile(filePath)
	f, err := os.Create(filePath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = fmt.Fprintf(f, "%09d\n", nonExpectedNumber)
	if err != nil {
		log.Println(err)
		return
	}

	expectedFileContent(t, err, filePath, nonExpectedNumber)

	numbers.NewFileWriter(ctx, numbersIn, filePath)
	fileEmpty(t, err, filePath)
}

func fileEmpty(t *testing.T, err error, filePath string) {
	time.Sleep(time.Second * 1)
	resultFile, err := os.Open(filePath)
	if err != nil {
		t.Fatal(err)
	}
	wResultFile := bufio.NewReader(resultFile)
	_, err = wResultFile.ReadString('\n')
	if err != nil && err.Error() != "EOF" {
		t.Fatal(err)
	}
	defer closeFile(resultFile)
}

func testFilePath(dir string) string {
	return dir + "/" + uuid.New().String() + ".log"
}

func cleanUpFile(filePath string) {
	err := os.Remove(filePath)
	if err != nil {
		log.Println(err)
	}
}

func expectedFileContent(t *testing.T, err error, filePath string, expectedNumber int) {
	time.Sleep(time.Second * 1)
	resultFile, err := os.Open(filePath)
	if err != nil {
		t.Fatal(err)
	}
	wResultFile := bufio.NewReader(resultFile)
	line, err := wResultFile.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	defer closeFile(resultFile)

	expectedString := strconv.Itoa(expectedNumber) + "\n"
	if line != expectedString {
		t.Fatal(fmt.Errorf("number should be: %s not %s", expectedString, expectedString))
	}
}

func closeFile(f *os.File) {
	if err := f.Close(); err != nil {
		log.Println(err)
	}
}
