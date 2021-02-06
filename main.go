package main

import "fmt"

func main() {

	foo, err := GetRaspberryPiIP()

	fmt.Println(err)
	fmt.Printf("%v", foo)
}
