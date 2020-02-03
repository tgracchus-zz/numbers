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

func NewNumbersProtocol(readTimeOut int) (TCPProtocol, chan bool) {
	terminate := make(chan bool)
	return func(ctx context.Context, c net.Conn) error {
		if checkIfTerminated(ctx) {
			return fmt.Errorf("connection: %s, terminated", c.RemoteAddr().String())
		}

		if err := c.SetReadDeadline(time.Now().Add(time.Duration(readTimeOut) * time.Second)); err != nil {
			return fmt.Errorf("connection: %s, SetReadDeadline failed: %s", c.RemoteAddr().String(), err)
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

		//numbersPipe <- number

		return nil
	}, terminate
}
