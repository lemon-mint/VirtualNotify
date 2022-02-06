package main

import (
	"flag"
	"log"
	"time"

	virtualnotify "github.com/lemon-mint/VirtualNotify"
)

var mode = flag.String("mode", "pub", "pub or sub")

func main() {
	flag.Parse()

	if *mode == "pub" {
		pub()
	} else if *mode == "sub" {
		sub()
	} else {
		panic("invalid mode")
	}
}

func pub() {
	vn := virtualnotify.New("test")

	err := vn.Publish("e1")
	if err != nil {
		panic(err)
	}
}

func sub() {
	vn := virtualnotify.New("test")

	err := vn.Subscribe("e1")
	if err != nil {
		panic(err)
	}

	go vn.Run(time.Millisecond * 500)

	for {
		v, err := vn.Next()
		if err != nil {
			panic(err)
		}
		log.Println("event:", v.Name)
	}
}
