package main

import ( 
    "github.com/bitfield/script"
    "os"
)

func main() {
    script.Echo("Hello Big " + os.Args[1] + "\n").Stdout()
}

