
package main

import ( 
    "fmt"
    "os"
)

func hello(name string) string {
	return fmt.Sprintf("Hello %s!", name)
}

func main() {
    fmt.Println(hello(os.Args[1]))
}

