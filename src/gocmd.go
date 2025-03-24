package main

import ( 
    "fmt"
    "path"
    "os"
)

func main() {
    fmt.Printf("ToPath: %s\n", path.Join(os.Args[1:]...))
}

