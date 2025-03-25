package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/fkmiec/goscript/util"
)

/*
Command-line driven program to compile bitfield/script and other go code into unix-like command pipelines.

- compile one-line pipeline of script commands into an executable (Include this src folder on the PATH so the command will be immediately usable)
- default the name of the command to "gocmd"
- optionally specify a unique name for the command
- optionally specify additional imports (e.g. import os if you need to pass arguments to the resulting command)
- optionally specify a file for the code. If so, assume the file is the entire code, including main function and imports.
- provide an option to spit out a skeletal template for a file
- provide an option to save the source in the project under the command name (ie. for name=FindFiles, src file will be <project>/src/FindFiles.go)
- provide an option to list previously compiled commands
- provide an option to output the full path to the source file previously saved to the project (so can edit in your favorite code editor)
- provide an option to output the path to the project folder
- provide an option to execute or run the code after compilation
- support use of shebang to immediately execute the command inline. Shebang invokes the command and passes the filename of the script as the
	first argument. So, "#!/usr/bin/env -S goscript -x -f" should effectively act the same as combining file and run flags on the command line. For the
	file option, always look for and strip out the shebang line if present.
*/

type Repl struct {
	Imports []string
	Code    string
}

var pkgMatcher *regexp.Regexp
var buf *bytes.Buffer

func assembleSourceFile(dir, code, imports string) *bytes.Buffer {
	//Help the user with imports when writing a one-liner goscript with the --code option.
	//
	//Lookup any references to packages listed in the util/imports.go file and
	// add to the imports if not already there explicitly. Enable use of shorter aliases.
	var formattedImports []string
	if len(imports) > 0 {
		theImports := strings.Split(imports, ",")
		for _, imp := range theImports {
			imp = fmt.Sprintf("\"%s\"", imp)
			formattedImports = append(formattedImports, imp)
		}
	}

	pkgMatcher = regexp.MustCompile(`(\w+)\.`) //match a type, field or function accessor (e.g. pkg.Type or struct.Field or struct.Function)
	matches := pkgMatcher.FindAllStringSubmatch(code, -1)
	for _, m := range matches {
		if len(m) > 0 {
			k := m[1]
			v := util.ImportsMap[k]

			if v != "" {
				//Check if the key matches the basename for the import. If so, use the import as is.
				//Otherwise, prepend the key as an alias for the package (e.g. "re" instead of "regexp")
				if filepath.Base(v) != k {
					v = fmt.Sprintf("%s \"%s\"", k, v) //e.g. re "regexp"
				} else {
					v = fmt.Sprintf("\"%s\"", v) //e.g. "regexp"
				}
				formattedImports = append(formattedImports, v)
			}
		}
	}

	repl := Repl{
		Imports: formattedImports,
		Code:    code,
	}

	buf = processTemplate(dir, repl)

	return buf
}

func readSourceFile(filename string) *bytes.Buffer {
	// Using bufio.Scanner to read line by line
	file, err := os.Open(filename)
	check(err)

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
		check(err)
	}

	err = scanner.Err() //; err != nil {
	check(err)

	return buf
}

func processTemplate(dir string, repl Repl) *bytes.Buffer {
	var tmplFile = dir + "/script.tmpl"
	tmpl, err := template.New("script.tmpl").ParseFiles(tmplFile)
	check(err)

	buf = bytes.NewBuffer([]byte{})
	err = tmpl.Execute(buf, repl)
	check(err)

	return buf
}

func writeSourceFile(filename string, buf *bytes.Buffer) bool {

	// Open the file for writing, creates it if it doesn't exist, or truncates if it exists.
	file, err := os.Create(filename)
	check(err)

	// Ensure the file is closed after the function returns.
	defer file.Close()

	// Write the buffer to the file
	_, err = buf.WriteTo(file)
	check(err)

	return true
}

func getProjectPath() string {
	executableDir := os.Getenv("GOSCRIPT_PROJECT_DIR")
	if executableDir != "" {
		isExist := checkFileExists(executableDir)
		if isExist {
			srcDir := executableDir + "/src"
			if !checkFileExists(srcDir) {
				os.Mkdir(srcDir, 0766)
			}
			binDir := executableDir + "/bin"
			if !checkFileExists(binDir) {
				os.Mkdir(binDir, 0766)
			}
		} else {
			fmt.Printf("Directory specified by GOSCRIPT_PROJECT_DIR not found: %s\n", executableDir)
			os.Exit(1)
		}
	} else {
		executablePath, err := os.Executable()
		check(err)
		executableDir = filepath.Dir(executablePath)
	}
	return executableDir
}

func getCommandList(dir string) []string {
	cmds := []string{}
	list, err := os.ReadDir(dir + "/bin")
	check(err)
	for _, entry := range list {
		if !entry.IsDir() {
			cmds = append(cmds, entry.Name())
		}
	}
	sort.Strings(cmds)
	return cmds
}

func recompileCommands(dir string) {
	commands := getCommandList(dir)
	var srcFilename, binFilename string
	for _, name := range commands {
		srcFilename = dir + "/src/" + name + ".go"
		binFilename = dir + "/bin/" + name
		compileBinary(dir, srcFilename, binFilename)
	}
}

func compileBinary(dir, srcFilename, binFilename string) {
	cmd := exec.Command("go", "build", "-o", binFilename, srcFilename)
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%v: %s\n", err, out)
	}
}

func checkFileExists(filePath string) bool {
	_, error := os.Stat(filePath)
	//return !os.IsNotExist(err)
	return !errors.Is(error, os.ErrNotExist)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {

	var name string
	var imports string
	var code string
	var inputFile string
	var listCommands bool
	var recompile bool
	var path string
	var printDir bool
	var printTemplate bool
	var execCode bool
	var printShebang bool

	flag.StringVar(&name, "name", "gocmd", "A name for your command. Defaults to gocmd.")
	flag.StringVar(&name, "n", "gocmd", "A name for your command. Defaults to gocmd.")
	flag.StringVar(&imports, "imports", "", "A comma-separated list of go packages to import. Not used with --file option.")
	flag.StringVar(&imports, "i", "", "A comma-separated list of go packages to import. Not used with --file option.")
	flag.StringVar(&code, "code", "", "The code of your command. Defaults to empty string.")
	flag.StringVar(&code, "c", "", "The code of your command. Defaults to empty string.")

	flag.StringVar(&inputFile, "file", "", "A go src file, complete with main function and imports. Alternative to --code and --imports options.")
	flag.StringVar(&inputFile, "f", "", "A go src file, complete with main function and imports. Alternative to --code and --imports options.")

	flag.StringVar(&path, "path", "", "Print the path to the source file specified, if exists in the project. Blank if not found.")
	flag.StringVar(&path, "p", "", "Print the path to the source file specified, if exists in the project. Blank if not found.")
	flag.BoolVar(&printDir, "dir", false, "Print the directory path to the project.")
	flag.BoolVar(&printDir, "d", false, "Print the directory path to the project.")
	flag.BoolVar(&printTemplate, "template", false, "Print a template go source file to stdout. After edits, use --file to compile with goscript.")
	flag.BoolVar(&printTemplate, "t", false, "Print a template go source file to stdout. After edits, use --file to compile with goscript.")

	flag.BoolVar(&printShebang, "bang", false, "Print the expected shebang line.")
	flag.BoolVar(&printShebang, "b", false, "Print the expected shebang line.")

	flag.BoolVar(&listCommands, "list", false, "Print the list of previously-compiled commands.")
	flag.BoolVar(&listCommands, "l", false, "Print the list of previously-compiled commands.")

	flag.BoolVar(&recompile, "recompile", false, "Recompile all existing source files in the project src directory.")

	flag.BoolVar(&execCode, "exec", false, "Execute the resulting binary.")
	flag.BoolVar(&execCode, "x", false, "Execute the resulting binary.")

	// Custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "goscript (see https://github.com/fkmiec/goscript)\n\n")
		fmt.Fprintf(os.Stderr, "The goscript command uses a dedicated Go module project and a Go template (script.tmpl from the repo) to compile Go scripts. The module project must have 'src' and 'bin' subfolders for your scripts and resulting binaries. The 'bin' folder must be on your PATH so that resulting binaries are immediately executable system-wide. The project directory is assumed to be wherever the binary is located. To use a different project location, set the environment variable GOSCRIPT_PROJECT_DIR.\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintln(os.Stderr, "  --code|-c string\n\tThe code of your command. Defaults to empty string.")
		fmt.Fprintln(os.Stderr, "  --imports|-i string\n\tA comma-separated list of go packages to import. Not used with --file option.")
		fmt.Fprintln(os.Stderr, "  --file|-f string\n\tA go src file, complete with main function and imports. Alternative to --code and --imports options.")
		fmt.Fprintln(os.Stderr, "  --name|-n string\n\tA name for your command. Defaults to gocmd.")
		fmt.Fprintln(os.Stderr, "  --list|-l\n\tPrint the list of previously-compiled commands.")
		fmt.Fprintln(os.Stderr, "  --recompile\n\tRecompile existing source files in the project src directory.")
		fmt.Fprintln(os.Stderr, "  --dir|-d\n\tPrint the directory path to the project.")
		fmt.Fprintln(os.Stderr, "  --path|-p\n\tPrint the path to the source file specified, if exists in the project. Blank if not found.")
		fmt.Fprintln(os.Stderr, "  --bang|-b\n\tPrint the expected shebang line.")
		fmt.Fprintln(os.Stderr, "  --template|-t\n\tPrint a template go source file to stdout. After edits, use --file to compile with goscript.")
		fmt.Fprintln(os.Stderr, "  --exec|-x\n\tExecute the resulting binary.")
		fmt.Fprintln(os.Stderr, "\nExample (Compile as 'hello'. Execute hello.):")
		fmt.Fprintf(os.Stderr, "  %s --code 'script.Echo(\" Hello World!\\n\").Stdout()' --name hello; hello\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "\nExample (Execute immediately.):")
		fmt.Fprintf(os.Stderr, "  %s --exec --code 'script.Echo(\" Hello World!\\n\").Stdout()'\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "\nExample shebang in 'myscript.go' file:")
		fmt.Fprintf(os.Stderr, "  (1) Add '#!/usr/bin/env -S %s' to the top of your go source file.\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "  (2) Set execute permission and type \"./myscript.go\" as you would with a shell script.\n")
	}

	//Shebang scenarios (Note any of these could also be straight commandline and not shebang):
	// (1) #!/usr/bin/env -S goscript -x -f <filename> <optionally more args> (handled as normal)
	// (2) #!/usr/bin/env -S goscript -x <filename> <optionally more args> (need to determine if first non-flag arg is a filename, and if so, set inputFile=arg[0])
	// (3) #!/usr/bin/env -S goscript <filename> <optionally more args> (need to determine if first non-flag arg is a filename, and if so, set inputFile=arg[0] and execCode=true)

	//The flag pkg expects non-flags to follow AFTER any flags given. However, shebang will make the filename the first arg.
	// So, before parsing the flags, check if first arg is a non-flag and an existing file.
	// If so, make it the inputFile and remove it from the os.Args array.
	// Beyond this one use case, we expect the user to follow convention and pass flags before non-flags.
	var nonFlagFirstArg bool
	if len(os.Args) > 1 {
		nonFlagFirstArg = checkFileExists(os.Args[1])
	}
	if nonFlagFirstArg {
		inputFile = os.Args[1]
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}

	flag.Parse()

	if nonFlagFirstArg && !execCode {
		execCode = true //Account for scenario 3, above.
	}

	var subprocessArgs []string
	if len(flag.Args()) > 0 {
		subprocessArgs = flag.Args()
	}

	//Get the path of the executable, which SHOULD BE the path to the project folder.
	//
	//NOTE: While it might typically make sense to install the binary in some other location,
	//  this binary exists to build code and so requires a project (in order
	//  to support modules, etc.). This could also be done locating the binary elsewhere and
	//  using an environment variable to specify the project directory used for compiling but
	//  this seems simpler.
	dir := getProjectPath()

	//--dir: Print the location of the project folder
	if printDir {
		fmt.Println(dir)
		return //Exit the program after printing the path
	}

	//--path: Print the location of the source file, if it exists, otherwise blank
	if path != "" {
		srcFile := dir + "/src/" + path + ".go"
		isFileExists := checkFileExists(srcFile)
		if isFileExists {
			//print the source file path
			fmt.Println(srcFile)
		}
		return //Exit the program after printing the path
	}

	//--bang: Print the shebang line to help the user who can't quite remember how it should go
	if printShebang {
		fmt.Println("#!/usr/bin/env -S " + os.Args[0])
		return //Exit the program after printing the shebang line
	}

	//--list: List existing commands
	if listCommands {
		cmds := getCommandList(dir)
		for _, cmd := range cmds {
			fmt.Printf("%s\n", cmd)
		}
		return //Exit the program after printing the list of commands
	}

	if recompile {
		recompileCommands(dir)
		return //Exit the program after recompiling existing commands
	}

	//--template: Print an empty template to give a starting point when creating an external source code file
	if printTemplate {
		buf = assembleSourceFile(dir, code, imports)
		fmt.Println("#!/usr/bin/env -S " + os.Args[0]) //Add the shebang line when printing a template
		_, err := buf.WriteTo(os.Stdout)
		check(err)
		return //Exit the program after printing the template
	}

	//--file: Handle a regular go source file (potentially with a shebang (#!) at the top)
	if inputFile != "" {
		buf = readSourceFile(inputFile)
		//--code: Handle typical one-liner code specified on command line
	} else if code != "" {
		buf = assembleSourceFile(dir, code, imports)
		//--name: Handle compiling a pre-existing source file located in the project/src folder
	} else if name != "gocmd" {
		srcFilename := dir + "/src/" + name + ".go"
		buf = readSourceFile(srcFilename)
		//(no options): Print usage and exit
	} else {
		flag.Usage()
		os.Exit(1)
	}

	//Save source and compile binary
	srcFilename := dir + "/src/" + name + ".go"
	binFilename := dir + "/bin/" + name

	writeSourceFile(srcFilename, buf)
	compileBinary(dir, srcFilename, binFilename)

	//Experiment
	if execCode {
		//Pass in any args intended for the subprocess
		cmd := exec.Command(binFilename, subprocessArgs...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			fmt.Fprintln(cmd.Stderr, err)
			os.Exit(1)
		}
		cmd.Wait()
	}
}
