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
	terminated := make(chan int)
	numbersIn := make(chan int)
	defer close(terminated)
	defer close(numbersIn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wireNumber := "098765432"
	expectedNumber, err := strconv.Atoi(wireNumber)
	if err != nil {
		t.Fatal(err)
	}
	sendData(t, client, wireNumber)
	go numbers.DefaultTCPController(ctx, server, numbersIn, terminated)

	number := <-numbersIn
	if expectedNumber != number {
		t.Fatal(fmt.Errorf("number should be: %d not %d", expectedNumber, number))
	}
}

func TestNumbersControllerNotNumber(t *testing.T) {
	server, client := net.Pipe()
	terminated := make(chan int)
	numbersIn := make(chan int)
	defer close(terminated)
	defer close(numbersIn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wireText := "IAMTEXT!!"
	sendData(t, client, wireText)
	go numbers.DefaultTCPController(ctx, server, numbersIn, terminated)
	ticker := time.NewTicker(time.Second)
	select {
	case number := <-numbersIn:
		t.Fatal(fmt.Errorf("non expected number:%d", number))
	case <-ticker.C:
	}
}

func TestNumbersControllerClosedConnection(t *testing.T) {
	server, _ := net.Pipe()
	terminated := make(chan int)
	numbersIn := make(chan int)
	defer close(terminated)
	defer close(numbersIn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := server.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = numbers.DefaultTCPController(ctx, server, numbersIn, terminated)
	if err == nil {
		t.Fatal("io: read/write on closed pipe was expected when parsing a text")
	}
}

func TestNumbersControllerTerminate(t *testing.T) {
	server, client := net.Pipe()
	terminated := make(chan int)
	numbersIn := make(chan int)
	defer close(numbersIn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sendData(t, client, "terminate")

	more := true
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		select {
		case _, more = <-terminated:
			wg.Done()
		case <-time.After(time.Second):
			wg.Done()
		}
	}()

	err := numbers.DefaultTCPController(ctx, server, numbersIn, terminated)
	if err == nil {
		t.Fatal("connection: pipe, terminated is expected")
	}
	wg.Wait()
	if more {
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
	numbersIn := make(chan int, 2)
	defer close(numbersIn)

	expectedNumber := 123456789
	numbersIn <- expectedNumber

	terminate := make(chan int)
	defer close(terminate)
	numberOut := numbers.NumberStore(10, []chan int{numbersIn}, terminate)

	expectNumber(numberOut, expectedNumber, t)
}

func TestNewNumberStoreTwoNumbers(t *testing.T) {
	numbersIn := make(chan int, 2)
	defer close(numbersIn)
	expectedNumber1 := 123456789
	expectedNumber2 := 987654321
	numbersIn <- expectedNumber1
	numbersIn <- expectedNumber2

	terminate := make(chan int)
	defer close(terminate)
	numberOut := numbers.NumberStore(10, []chan int{numbersIn}, terminate)

	expectNumber(numberOut, expectedNumber1, t)
	expectNumber(numberOut, expectedNumber2, t)
}

func TestNewNumberStoreDeduplicated(t *testing.T) {
	numbersIn := make(chan int, 2)
	defer close(numbersIn)
	expectedNumber := 123456789

	numbersIn <- expectedNumber
	numbersIn <- expectedNumber

	terminate := make(chan int)
	defer close(terminate)
	numberOut := numbers.NumberStore(10, []chan int{numbersIn}, terminate)

	expectNumber(numberOut, expectedNumber, t)
	numberNotExpected(numberOut, t)
}

func TestNewNumberStoreCloseInChannel(t *testing.T) {
	numbersIn := make(chan int)
	close(numbersIn)

	terminate := make(chan int)
	defer close(terminate)
	numberOut := numbers.NumberStore(10, []chan int{numbersIn}, terminate)
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
	numbersIn := make(chan int, 2)
	expectedNumber := 123456789
	numbersIn <- expectedNumber

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	filePath := testFilePath(dir)
	defer cleanUpFile(filePath)

	done := numbers.FileWriter(numbersIn, filePath)
	close(numbersIn)
	select {
	case _, more := <-done:
		if !more {
			expectedFileContent(t, err, filePath, expectedNumber)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout while waiting for a response in numberOut")
	}
}

func TestNewFileWriterOverwriteWhenStarted(t *testing.T) {
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

	numbers.FileWriter(numbersIn, filePath)
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
