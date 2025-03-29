package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
)

func main() {
	url := "https://api.my-ip.io/v2/ip.txt"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()

	s := bufio.NewScanner(resp.Body)
	for s.Scan() {
		fmt.Println(s.Text())
	}

}
