package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

/*
Command-line driven program to compile bitfield/script and other go code into unix-like command pipelines.

- compile one-line pipeline of script commands into an executable (Include this src folder on the PATH so the command will be immediately usable)
- default the name of the command to "gocmd"
- optionally specify a unique name for the command
- optionally specify additional imports (e.g. import os if you need to pass arguments to the resulting command)

- optionally specify a file for the code. If so, assume the file is the entire code, including main function and imports.
- provide an option to spit out a skeleton for a file
- provide an option to save the source in the project under the command name (ie. for name=FindFiles, src file will be <project>/src/FindFiles.go)
- provide an option to output the full path to the source file previously saved to the project (so can edit in your favorite code editor)
- provide an option to output the path to the project folder
- support use of shebang to immediately execute the command inline. Shebang invokes the command and passes the filename of the script as the
	first argument. So, "#!/<dir>/goscripts -r -f" should effectively act the same as combining file and run flags on the command line. For the
	file option, always look for and strip out the shebang line if present.
- provide an option to execute or run the code after compilation
- provide an option to "go get" a required package (Arguably not necessary. Just explain in README.md that external packages might require a go get first.)

Implementation:
- Use go template to populate a skeleton main function
- Execute go build command on the filled in template from this code
- Execute the binary if flag to run is provided
*/

type Repl struct {
	Imports []string
	Code    string
}

var buf *bytes.Buffer

func readSourceFile(filename string) *bytes.Buffer {
	// Using bufio.Scanner to read line by line
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var line string
	buf = bytes.NewBuffer([]byte{})
	for scanner.Scan() {

		line = scanner.Text()
		//strip out the shebang if present
		if strings.HasPrefix(line, "#!") {
			continue
		}
		_, err := buf.WriteString(line + "\n")
		if err != nil {
			log.Fatal(err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return buf
}

func processTemplate(repl Repl) *bytes.Buffer {
	var tmplFile = "script.tmpl"
	tmpl, err := template.New(tmplFile).ParseFiles(tmplFile)
	if err != nil {
		panic(err)
	}

	buf = bytes.NewBuffer([]byte{})
	err = tmpl.Execute(buf, repl)
	if err != nil {
		panic(err)
	}
	return buf
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

func getProjectPath() string {
	executablePath, err := os.Executable()
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}

	executableDir := filepath.Dir(executablePath)
	return executableDir
}

func main() {

	var name string
	var imports string
	var code string
	var inputFile string
	var saveSource bool
	var printPath bool
	var printTemplate bool
	var execCode bool

	flag.StringVar(&name, "name", "gocmd", "A name for your command (a string). Defaults to gocmd.")
	flag.StringVar(&name, "n", "gocmd", "A name for your command (a string). Defaults to gocmd.")
	flag.StringVar(&imports, "imports", "", "A comma-separated list of go packages (a string)")
	flag.StringVar(&imports, "i", "", "A comma-separated list of go packages (a string)")
	flag.StringVar(&code, "code", "fmt.Println(\"Your Code Here!\")", "Your code to run (a string)")
	flag.StringVar(&code, "c", "fmt.Println(\"Your Code Here!\")", "Your code to run (a string)")

	flag.StringVar(&inputFile, "file", "", "A go src file, complete with main function and imports (a string)")
	flag.StringVar(&inputFile, "f", "", "A go src file, complete with main function and imports (a string)")
	flag.BoolVar(&saveSource, "savesrc", false, "A boolean flag. Saves a source file <cmd name>.go to the project. Defaults to false if absent.")
	flag.BoolVar(&saveSource, "s", false, "A boolean flag. Saves a source file <cmd name>.go to the project. Defaults to false if absent.")

	flag.BoolVar(&printPath, "path", false, "A boolean flag. If true, prints the path to the source file, if name provided, or project. Defaults to false.")
	flag.BoolVar(&printPath, "p", false, "A boolean flag. If true, prints the path to the source file, if name provided, or project. Defaults to false.")
	flag.BoolVar(&printTemplate, "template", false, "A boolean flag. If true, prints the template to stdout to facilitate writing a source file. Defaults to false.")
	flag.BoolVar(&printTemplate, "t", false, "A boolean flag. If true, prints the template to stdout to facilitate writing a source file. Defaults to false.")

	flag.BoolVar(&execCode, "exec", false, "A boolean flag. If true, execute the resulting binary. Defaults to false.")
	flag.BoolVar(&execCode, "x", false, "A boolean flag. If true, execute the resulting binary. Defaults to false.")

	flag.Parse()

	//Get the path of the executable, which we assume is the project folder.
	//NOTE: While it might typically make sense to install the binary in some other location,
	//  for this project that aims to compile other go code within the same project (in order
	//  to support modules, etc.), we either make this assumption or require a PATH_TO_GOSCRIPTS_PROJECT
	//  environment variable to make this work on the user's system.
	dir := getProjectPath()

	if printPath {
		if name == "gocmd" {
			//print the project path
			fmt.Println(dir)
		} else {
			//print the source file path
			fmt.Println(dir + "/src/" + name + ".go")
		}
		return //Exit the program after printing the path
	}

	//Handle typical one-liner code specified on command line
	if inputFile == "" {
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

		buf = processTemplate(repl)

		//Helper code prints an empty template to give a starting point when creating an external file manually
		if printTemplate {
			_, err := buf.WriteTo(os.Stdout)
			if err != nil {
				fmt.Println("Error writing to stdout:", err)
				return
			}
			return //exit the program after printing the template
		}

		//Handle a regular go source file (potentially with a shebang (#!) at the top)
	} else {
		buf = readSourceFile(inputFile)
	}

	//Save source and compile binary
	srcFilename := dir + "/src/tmp.go"
	if saveSource {
		srcFilename = dir + "/src/" + name + ".go"
	}
	binFilename := dir + "/bin/" + name

	writeFile(srcFilename, buf)
	cmd := exec.Command("go", "build", "-o", binFilename, srcFilename)

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%v: %s\n", err, out)
	}

	//If flag to run the code was given, then execute it
	if execCode {
		cmd := exec.Command(binFilename)

		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("%v: %s\n", err, out)
		} else {
			fmt.Printf("%s\n", out)
		}
	}
}
