package tcp

import (
	"fmt"
	"log"
	"time"
)

func ExampleShaker() {
	s := Shaker{}
	if err := s.Init(); err != nil {
		log.Fatal("Shaker init failed:", err)
	}

	timeout := time.Second * 1
	err := s.Test("google.com:80", timeout)
	switch err {
	case ErrTimeout:
		fmt.Println("Connect to Google timeout")
	case nil:
		fmt.Println("Connect to Google succeded")
	default:
		fmt.Println("Connect to Google failed:", err)
	}
}
