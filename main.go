package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"text/template"

	"github.com/fkmiec/goscript/util"
)

type Repl struct {
	Imports []string
	Code    string
}

var version string = "goscript v1.2.1"
var projectDir string
var pkgMatcher *regexp.Regexp
var buf *bytes.Buffer

func assembleSourceFile(code string) *bytes.Buffer {
	//If user wants to put main function body in a file and read it in, rather than cumbersome command line, we can do that.
	if checkFileExists(code) {
		buf = readSourceFile(code)
		code = buf.String()
	}
	//Automate imports when writing a one-liner goscript with the --code option.

	//Lookup any references to packages listed in the util/imports.go file and
	// add to the imports if not already there explicitly. Enable use of shorter aliases.
	var formattedImports []string

	//Read in any additional import mappings from imports.json file in project directory
	userImports := readUserImports()
	if userImports != nil {
		for key, value := range userImports {
			util.ImportsMap[key] = value
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
				//Ensure we don't duplicate any imports
				if !slices.Contains(formattedImports, v) {
					formattedImports = append(formattedImports, v)
				}
			}
		}
	}

	repl := Repl{
		Imports: formattedImports,
		Code:    code,
	}

	buf = processTemplate(repl)
	formatted, err := formatCode(buf.Bytes())
	check(err)
	buf.Reset()
	buf.Write(formatted)
	return buf
}

func formatCode(bytes []byte) ([]byte, error) {
	formatted, err := format.Source(bytes)
	check(err)
	return formatted, nil
}

func readUserImports() map[string]string {
	var userImports map[string]string
	filename := projectDir + "/imports.json"
	if checkFileExists(filename) {
		file, err := os.Open(filename)
		check(err)
		defer file.Close()

		byteValue, _ := io.ReadAll(file)
		json.Unmarshal([]byte(byteValue), &userImports)
	}
	return userImports
}

func writeUserImports(userImports map[string]string) {
	filename := projectDir + "/imports.json"
	jsonData, err := json.MarshalIndent(userImports, "", "    ") // Use MarshalIndent for pretty printing
	check(err)
	err = os.WriteFile(filename, jsonData, 0644)
	check(err)
}

func goGet(pkgName string) {
	cmd := exec.Command("go", "get", pkgName)
	cmd.Dir = projectDir

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%v: %s\n", err, out)
	}

	//Add pkgName to imports.json file
	pkgAlias := filepath.Base(pkgName)
	userImports := readUserImports()
	if userImports == nil {
		userImports = make(map[string]string)
	}
	userImports[pkgAlias] = pkgName
	writeUserImports(userImports)
}

func goTidy() {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectDir

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%v: %s\n", err, out)
	}
}

func editCommand(cmd string) {
	srcFilename := projectDir + "/src/" + cmd + ".go"
	if checkFileExists(srcFilename) {
		editor := os.Getenv("GOSCRIPT_EDITOR")
		if editor == "" {
			editor = os.Getenv("EDITOR")
			if editor == "" {
				fmt.Printf("The --edit option requires environment variable GOSCRIPT_EDITOR or EDITOR to be defined.")
				return
			}
		}
		cmd := exec.Command(editor, srcFilename)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			fmt.Fprintln(cmd.Stderr, err)
			os.Exit(1)
		}
		cmd.Wait()
	} else {
		fmt.Printf("File not found in <project>/src directory for %s\n", cmd)
		return
	}
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

func processTemplate(repl Repl) *bytes.Buffer {
	var tmplFile = projectDir + "/script.tmpl"
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

func copyFile(orig string, dest string) {
	origFile, err := os.Open(orig)
	check(err)
	defer origFile.Close()

	destFile, err := os.Create(dest)
	check(err)
	defer destFile.Close()

	_, err = io.Copy(destFile, origFile)
	check(err)

	err = os.Chmod(dest, 0766)
	check(err)
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

func getSourceList() []string {
	cmds := []string{}
	list, err := os.ReadDir(projectDir + "/src")
	check(err)
	for _, entry := range list {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			cmds = append(cmds, entry.Name())
		}
	}
	sort.Strings(cmds)
	return cmds
}

// Soft delete. Renames source file without .go extension so it will be ignored. Removes binary.
func deleteCommand(cmd string) {
	sansGoExt := projectDir + "/src/" + cmd
	srcFilename := sansGoExt + ".go"
	binFilename := projectDir + "/bin/" + cmd
	err := os.Rename(srcFilename, sansGoExt)
	check(err)
	err = os.Remove(binFilename)
	check(err)
	goTidy() //run go mod tidy to keep go.mod file current when you remove sources
}

// Soft delete. Renames source file without .go extension so it will be ignored. Removes binary.
func restoreCommand(cmd string) {
	sansGoExt := projectDir + "/src/" + cmd
	srcFilename := sansGoExt + ".go"
	binFilename := projectDir + "/bin/" + cmd
	err := os.Rename(sansGoExt, srcFilename)
	check(err)
	compileBinary(srcFilename, binFilename)
}

func recompileCommands() {
	commands := getSourceList()
	var srcFilename, binFilename string
	for _, name := range commands {
		srcFilename = projectDir + "/src/" + name
		binFilename = projectDir + "/bin/" + name[:len(name)-3] //removes .go from binary filename
		compileBinary(srcFilename, binFilename)
	}
}

func compileBinary(srcFilename, binFilename string) {
	cmd := exec.Command("go", "build", "-o", binFilename, srcFilename)
	cmd.Dir = projectDir

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%v: %s\n", err, out)
	}
}

func createNewProject(dir string) {
	if dir == "help" {
		fmt.Printf("To use the --setup option to create a goscript project:\n")
		fmt.Printf("Run '%s --setup <project name>'\n", os.Args[0])
		fmt.Printf("Goscript will:\n")
		fmt.Printf("  a. Create the project directory\n")
		fmt.Printf("  b. Run go mod init <project>\n")
		fmt.Printf("  c. Run 'go get github.com/bitfield/script'\n")
		fmt.Printf("  d. Create 'src' and 'bin' subdirectories in the project\n")
		fmt.Printf("  e. Add the required Go template file 'script.tmpl'\n")
		fmt.Printf("  f. Print out instructions to set GOSCRIPT_PROJECT_DIR and add GOSCRIPT_PROJECT_DIR/bin to the PATH\n")
		return
	}
	projectDir = dir
	isAbsolute := filepath.IsAbs(dir)
	if !isAbsolute {
		pwd, err := os.Getwd()
		if err != nil {
			fmt.Println(err)
			return
		}
		projectDir = pwd + "/" + dir
	}

	fmt.Printf("Absolute path: %s\n", projectDir)

	//Create project directory if not exist
	if !checkFileExists(projectDir) {
		os.Mkdir(projectDir, 0766)
	}

	//Run go mod init <basename>
	projectName := filepath.Base(projectDir)
	cmd := exec.Command("go", "mod", "init", projectName)
	cmd.Dir = projectDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%v: %s\n", err, out)
	}

	//Run go get github.com/bitfield/script
	cmd = exec.Command("go", "get", "github.com/bitfield/script")
	cmd.Dir = projectDir
	out, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%v: %s\n", err, out)
	}

	//Create 'src' and 'bin' subdirectories
	srcDir := projectDir + "/src"
	os.Mkdir(srcDir, 0766)
	binDir := projectDir + "/bin"
	os.Mkdir(binDir, 0766)

	//Write script.tmpl file
	// Open the file for writing, creates it if it doesn't exist, or truncates if it exists.
	filename := projectDir + "/script.tmpl"
	file, err := os.Create(filename)
	check(err)
	defer file.Close()
	file.WriteString("package main\n\nimport ( {{range .Imports}}\n\t{{.}}{{ end }}\n)\n\nfunc main() {\n\t{{.Code}}\n}\n")

	//Print instructions to set environment variable GOSCRIPT_PROJECT_DIR and add GOSCRIPT_PROJECT_DIR/bin to PATH
	fmt.Printf("Created project %s at %s\n", projectName, projectDir)
	fmt.Printf("To complete setup:\n")
	fmt.Printf("\t1. Set environment variable GOSCRIPT_PROJECT_DIR=%s\n", projectDir)
	fmt.Printf("\t2. Add %s to your PATH environment variable.\n", binDir)
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
	var toEdit string
	var toCat string
	var toExport string
	var binToExport string
	var toDelete string
	var toRestore string
	var code string
	var inputFile string
	var listCommands bool
	var recompile bool
	var setupProject string
	var toGoGet string
	var path string
	var printDir bool
	var printTemplate bool
	var execCode bool
	var printShebang bool
	var printVersion bool

	flag.StringVar(&name, "name", "", "A name for your command.")
	flag.StringVar(&name, "n", "", "A name for your command.")
	flag.StringVar(&toCat, "cat", "", "Prints the named script to stdout. The source and binary remain in the project.")
	flag.StringVar(&toExport, "export", "", "Exports the named script to stdout with shebang added and removes source and binary from project.")
	flag.StringVar(&binToExport, "export-bin", "", "Exports the named binary to local directory and removes source and binary from project.")
	flag.StringVar(&toEdit, "edit", "", "Edit the named command in the editor specified by environment variable GOSCRIPT_EDITOR or EDITOR.")
	flag.StringVar(&toEdit, "e", "", "Edit the named command in the editor specified by environment variable GOSCRIPT_EDITOR or EDITOR.")
	flag.StringVar(&code, "code", "", "The code of your command. Defaults to empty string.")
	flag.StringVar(&code, "c", "", "The code of your command. Defaults to empty string.")

	flag.StringVar(&inputFile, "file", "", "A go src file, complete with main function and imports. Alternative to --code and --imports options.")
	flag.StringVar(&inputFile, "f", "", "A go src file, complete with main function and imports. Alternative to --code and --imports options.")
	flag.StringVar(&toDelete, "delete", "", "Delete the specified compiled command. Removes .go extension from source file so it can be restored.")
	flag.StringVar(&toRestore, "restore", "", "Restore a command after delete or export operation. Restores .go extension to the source file and recompiles.")

	flag.StringVar(&path, "path", "", "Print the path to the source file specified, if exists in the project. Blank if not found.")
	flag.StringVar(&path, "p", "", "Print the path to the source file specified, if exists in the project. Blank if not found.")
	flag.BoolVar(&printDir, "dir", false, "Print the directory path to the project.")
	flag.BoolVar(&printDir, "d", false, "Print the directory path to the project.")
	flag.BoolVar(&printTemplate, "template", false, "Print a template go source file to stdout. After edits, use --file to compile with goscript.")
	flag.BoolVar(&printTemplate, "t", false, "Print a template go source file to stdout. After edits, use --file to compile with goscript.")

	flag.BoolVar(&printShebang, "bang", false, "Print the expected shebang line.")
	flag.BoolVar(&printShebang, "b", false, "Print the expected shebang line.")

	flag.BoolVar(&listCommands, "list", false, "Print the list of existing commands.")
	flag.BoolVar(&listCommands, "l", false, "Print the list of existing commands.")

	flag.StringVar(&setupProject, "setup", "", "A name or absolute path. Creates a module project to be used by goscript. If no name is given, prints setup instructions.")
	flag.BoolVar(&recompile, "recompile", false, "Recompile all existing source files in the project src directory.")
	flag.StringVar(&toGoGet, "goget", "", "Go get an external package (not part of stdlib) to pull into the project.")
	flag.StringVar(&toGoGet, "g", "", "Go get an external package (not part of stdlib) to pull into the project.")

	flag.BoolVar(&execCode, "exec", false, "Execute the resulting binary.")
	flag.BoolVar(&execCode, "x", false, "Execute the resulting binary.")

	flag.BoolVar(&printVersion, "version", false, "Print the goscript version.")
	flag.BoolVar(&printVersion, "v", false, "Print the goscript version.")

	// Custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s (see https://github.com/fkmiec/goscript)\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintln(os.Stderr, "  --code|-c string\n\tThe code of your command or the name of a file containing the body of the main function.")
		fmt.Fprintln(os.Stderr, "  --file|-f string\n\tA go src file, complete with main function and imports. Alternative to --code.")
		fmt.Fprintln(os.Stderr, "  --exec|-x\n\tExecute the resulting binary.")
		fmt.Fprintln(os.Stderr, "  --name|-n string\n\tA name for your command. The code will be saved to the project src directory with that name.")
		fmt.Fprintln(os.Stderr, "  --edit|-e string\n\tEdit the named command in the editor specified by environment variable GOSCRIPT_EDITOR or EDITOR.")
		fmt.Fprintln(os.Stderr, "  --template|-t\n\tPrint a template go source file to stdout, or to the project src directory if --name provided.")
		fmt.Fprintln(os.Stderr, "  --list|-l\n\tPrint the list of existing commands.")
		fmt.Fprintln(os.Stderr, "  --path|-p string\n\tPrint the path to the source file specified, if exists in the project. Blank if not found.")
		fmt.Fprintln(os.Stderr, "  --cat string\n\tPrints the named script to stdout. The source and binary remain in the project.")
		fmt.Fprintln(os.Stderr, "  --export string\n\tExports the named script to stdout with shebang added and removes source and binary from project.")
		fmt.Fprintln(os.Stderr, "  --export-bin string\n\tExports the named binary to the local directory and removes source and binary from project.")
		fmt.Fprintln(os.Stderr, "  --delete string\n\tDelete the specified compiled command. Removes .go extension from source file so it remains recoverable.")
		fmt.Fprintln(os.Stderr, "  --restore string\n\tRestore a command after delete or export operation. Restores .go extension to the source file and recompiles.")
		fmt.Fprintln(os.Stderr, "  --goget|-g string\n\tGo get an external package (not part of stdlib) to pull into the project.")
		fmt.Fprintln(os.Stderr, "  --recompile\n\tRecompile existing source files in the project src directory.")
		fmt.Fprintln(os.Stderr, "  --setup\n\tA name, absolute path or 'help'. Creates a module project to be used by goscript. If 'help', prints setup instructions.")
		fmt.Fprintln(os.Stderr, "  --dir|-d\n\tPrint the directory path to the project.")
		fmt.Fprintln(os.Stderr, "  --bang|-b\n\tPrint the expected shebang line.")
		fmt.Fprintln(os.Stderr, "  --version|-v\n\tPrint the goscript version.")
		fmt.Fprintln(os.Stderr, "\nExample (Compile as 'hello'. Execute hello.):")
		fmt.Fprintf(os.Stderr, "  %s --code 'script.Echo(\"Hello World!\\n\").Stdout()' --name hello; hello\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "\nExample (Execute immediately.):")
		fmt.Fprintf(os.Stderr, "  %s --exec --code 'script.Echo(\"Hello World!\\n\").Stdout()'\n", os.Args[0])
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

	//Get the project path (either the location of the executable or as specified by GOSCRIPT_PROJECT_DIR).
	projectDir = getProjectPath()

	//--version: Print the version of goscript
	if printVersion {
		fmt.Println(version)
		return //Exit the program after printing the version
	}

	//--dir: Print the location of the project folder
	if printDir {
		fmt.Println(projectDir)
		return //Exit the program after printing the path
	}

	//--path: Print the location of the source file, if it exists, otherwise blank
	if path != "" {
		srcFile := projectDir + "/src/" + path + ".go"
		isFileExists := checkFileExists(srcFile)
		if isFileExists {
			//print the source file path
			fmt.Println(srcFile)
		}
		return //Exit the program after printing the path
	}

	//--setup: Create new goscript project. If no project name or path given, prints setup instructions.
	if setupProject != "" {
		createNewProject(setupProject)
		return //Exit the program after setting up project or printing instructions.
	}

	//--bang: Print the shebang line to help the user who can't quite remember how it should go
	if printShebang {
		fmt.Println("#!/usr/bin/env -S " + os.Args[0])
		return //Exit the program after printing the shebang line
	}

	//--list: List existing commands
	if listCommands {
		cmds := getSourceList() //Assumes binary list is same. Not true if template files that were never compiled, but should be rare.
		for _, cmd := range cmds {
			fmt.Printf("%s\n", cmd[:len(cmd)-3]) //Remove the .go extension.
		}
		return //Exit the program after printing the list of commands
	}

	//--goget: Execute a go get <pkg> to bring external package into project
	if toGoGet != "" {
		goGet(toGoGet)
		return //Exit after go get package
	}

	//--recompile: Recompile existing sources
	if recompile {
		recompileCommands()
		return //Exit the program after recompiling existing commands
	}

	//--template: Print an empty template to give a starting point when creating a new source code file
	if printTemplate {
		buf = assembleSourceFile(code)
		if name != "" {
			srcFilename := projectDir + "/src/" + name + ".go"
			writeSourceFile(srcFilename, buf)
			fmt.Printf("Source file written to: %s\n", srcFilename)
			return
		} else {
			fmt.Println("#!/usr/bin/env -S " + os.Args[0]) //Add the shebang line when printing a template
			_, err := buf.WriteTo(os.Stdout)
			check(err)
			return //Exit the program after printing the template
		}
	}

	//--edit: Edit the source code from the named command using GOSCRIPT_EDITOR or EDITOR. If neither defined, then print help message.
	if toEdit != "" {
		editCommand(toEdit)
		return //Exit the program after exporting
	}

	//--cat: Print the source code from the named command to stdout.
	if toCat != "" {
		srcFilename := projectDir + "/src/" + toCat + ".go"
		buf = readSourceFile(srcFilename)
		//fmt.Println("#!/usr/bin/env -S " + os.Args[0]) //Add the shebang line when exporting a source file (assumption is outside project it will be a shebang script)
		_, err := buf.WriteTo(os.Stdout)
		check(err)
		return //Exit the program after printing
	}

	//--export: Print the source code from the named command to stdout.
	// Executes --delete option as well (see below)
	if toExport != "" {
		srcFilename := projectDir + "/src/" + toExport + ".go"
		buf = readSourceFile(srcFilename)
		fmt.Println("#!/usr/bin/env -S " + os.Args[0]) //Add the shebang line when exporting a source file (assumption is outside project it will be a shebang script)
		_, err := buf.WriteTo(os.Stdout)
		check(err)
		deleteCommand(toExport)
		return //Exit the program after exporting
	}

	//--export-bin: Copy the binary to the local directory.
	// Executes --delete option as well (see below)
	if binToExport != "" {
		binFilename := projectDir + "/bin/" + binToExport
		copyFile(binFilename, binToExport)
		deleteCommand(binToExport)
		return //Exit the program after exporting
	}

	//--delete: Deletes the named binary. Renames the named source file without .go extension so it remains recoverable.
	if toDelete != "" {
		deleteCommand(toDelete)
		return //Exit the program after deleting
	}

	//--restore: Restores the named binary that was previously deleted or exported. Adds the .go extension back to the source file and recompiles.
	if toRestore != "" {
		restoreCommand(toRestore)
		return //Exit the program after restoring
	}

	//--file: Handle a regular go source file (potentially with a shebang (#!) at the top)
	if inputFile != "" {
		buf = readSourceFile(inputFile)
		//--code: Handle typical one-liner code specified on command line
	} else if code != "" {
		buf = assembleSourceFile(code)
		//--name: Handle compiling a pre-existing source file located in the project/src folder
	} else if name != "" {
		srcFilename := projectDir + "/src/" + name + ".go"
		buf = readSourceFile(srcFilename)
		//(no options): Print usage and exit
	} else {
		flag.Usage()
		os.Exit(1)
	}

	//Save source and compile binary
	if name == "" {
		name = "gocmd"
	}
	srcFilename := projectDir + "/src/" + name + ".go"
	binFilename := projectDir + "/bin/" + name

	writeSourceFile(srcFilename, buf)
	compileBinary(srcFilename, binFilename)

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
