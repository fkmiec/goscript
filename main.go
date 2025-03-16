package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

/*
Command-line driven program to compile bitfield/script and other go code into unix-like command pipelines.

- compile one-line pipeline of script commands into an executable (Include this src folder on the PATH so the command will be immediately usable)
- default the name of the command to "gocmd"
- optionally specify a unique name for the command
- optionally specify additional imports (e.g. import os if you need to pass arguments to the resulting command)
- support use of shebang to immediately execute the command inline (How? Is it necessary?)

Implementation:
- Use go template to populate a skeleton main function
- Execute go build command on the filled in template from this code
- Alternatively execute go run on the filled in template from this code (if shebang for go run works, then this would support indirect shebang)
*/

type Repl struct {
	Imports []string
	Code    string
}

func writeFile(filename string, buf *bytes.Buffer) bool {
	// Open the file for writing, creates it if it doesn't exist, or truncates if it exists.
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return false
	}
	// Ensure the file is closed after the function returns.
	defer file.Close()

	// Write the buffer to the file
	_, err = buf.WriteTo(file)
	if err != nil {
		fmt.Println("Error writing buffer to file:", err)
		return false
	}

	return true
}

func main() {

	var name string
	var imports string
	var code string
	flag.StringVar(&name, "name", "gocmd", "A name for your command (a string). Defaults to tmpcmd.")
	flag.StringVar(&name, "n", "gocmd", "A name for your command (a string). Defaults to tmpcmd.")
	flag.StringVar(&imports, "imports", "", "A comma-separated list of go packages (a string)")
	flag.StringVar(&imports, "i", "", "A comma-separated list of go packages (a string)")
	flag.StringVar(&code, "code", "fmt.Println(\"Your Code Here!\")", "Your code to run (a string)")
	flag.StringVar(&code, "c", "fmt.Println(\"Your Code Here!\")", "Your code to run (a string)")

	flag.Parse()

	//Default use case for this little script builder is the use of bitfield/script.
	//So, we try to include that import if not given explicitly.
	if strings.Contains(code, "script.") {
		if !strings.Contains(imports, "github.com/bitfield/script") {
			if len(imports) > 0 {
				imports += ",github.com/bitfield/script"
			} else {
				imports = "github.com/bitfield/script"
			}
		}
	}

	theImports := strings.Split(imports, ",")

	repl := Repl{
		Imports: theImports,
		Code:    code,
	}

	var tmplFile = "script.tmpl"
	tmpl, err := template.New(tmplFile).ParseFiles(tmplFile)
	if err != nil {
		panic(err)
	}

	buf := bytes.NewBuffer([]byte{})
	err = tmpl.Execute(buf, repl)
	if err != nil {
		panic(err)
	}

	writeFile("./tmp.go", buf)
	cmd := exec.Command("go", "build", "-o", name, "./tmp.go")

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%v: %s\n", err, out)
	}

}
