package numbers

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func NewNumbersProtocol(numbersPipe chan int) TCPProtocol {
	return func(c net.Conn) error {
		if err := c.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			return fmt.Errorf("client: %s, SetReadDeadline failed: %s", c.RemoteAddr().String(), err)
		}
		reader := bufio.NewReader(c)
		data, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		data = strings.TrimSuffix(data, "\n")
		if data == "terminate" {
			log.Printf("client: %s, terminating ", c.RemoteAddr().String())
			return nil
		}

		if len(data) != 9 {
			return fmt.Errorf("client: %s, no 9 char length string %s", c.RemoteAddr().String(), data)
		}

		number, err := strconv.Atoi(data)
		if err != nil {
			log.Println(err)
			return err
		}

		log.Printf("client: %s, number: %d", c.RemoteAddr().String(), number)
		//numbersPipe <- number

		return nil
	}
}
