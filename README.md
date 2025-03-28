# goscript

**Goscript** is a utility to make it easier to use Go as a scripting language. It was initially inspired by and implemented to leverage https://github.com/bitfield/script, a Go package aimed at bringing unix-like piped commands to Go.

The Go compiler is fast enough that using Go for scripting tasks can be a good choice. There are challenges, however. The `go run` command works to build and execute go code that relies on the stdlib well enough, but adding any third-party packages involves complications with go modules and/or the GOPATH and can quickly become confusing. There is no support for running short bits of code directly on the command line nor using a shebang at the top of a file to execute like a script. Also, managing your GOPATH and dependencies can be complicated and there is no help with organizing your scripts so they are easily found, updated and applied wherever you need them.  

Enter **Goscript**.  

- [goscript](#goscript)
  - [Features](#features)
  - [Philosophy](#philosophy)
  - [How It Works](#how-it-works)
  - [Install](#install)
    - [Option 1 - Clone or Fork This Repo](#option-1---clone-or-fork-this-repo)
    - [Option 2 - Go Install and goscript --setup](#option-2---go-install-and-goscript---setup)
  - [Usage](#usage)
  - [Examples](#examples)
    - [Compile and Execute in Two Steps](#compile-and-execute-in-two-steps)
    - [Compile and Execute in One Step with --exec](#compile-and-execute-in-one-step-with---exec)
    - [Name the Command for Repeat Use](#name-the-command-for-repeat-use)
    - [Required Imports Added Automatically](#required-imports-added-automatically)
    - [Optionally Use a File with --code](#optionally-use-a-file-with---code)
    - [Use --file to Pass a Source File](#use---file-to-pass-a-source-file)
    - [Shebang (Linux and Mac only)](#shebang-linux-and-mac-only)
    - [List Saved Commands](#list-saved-commands)
    - [Use --edit Option to Edit a Command's Source in Context of the Project](#use---edit-option-to-edit-a-commands-source-in-context-of-the-project)
    - [Use --cat Option to Print a Command's Source to Stdout](#use---cat-option-to-print-a-commands-source-to-stdout)
    - [Use --export Option to Export a Command's Source and Remove the Command from the Project](#use---export-option-to-export-a-commands-source-and-remove-the-command-from-the-project)
    - [Use --export-bin Option to Export a Command's Binary to the Current Directory and Remove it From the Project](#use---export-bin-option-to-export-a-commands-binary-to-the-current-directory-and-remove-it-from-the-project)
    - [Use --delete Option to "Soft Delete" a Command](#use---delete-option-to-soft-delete-a-command)
    - [Get Path to Project (support project maintenance)](#get-path-to-project-support-project-maintenance)
    - [Get Path to Source File (support editing)](#get-path-to-source-file-support-editing)
    - [Recompile Existing Commands](#recompile-existing-commands)
    - [Pipe Goscript Commands Together With Unix Commands](#pipe-goscript-commands-together-with-unix-commands)

## Features

* Execute simple go code directly on the command line
* Write go source files anywhere in your filesystem and execute them like a script with shebang
* Compile go code into named binaries for repeated use. Build up a library of custom commands. 
* Everything in one dedicated project, accessible system-wide through the **goscript** command. 
* Options to generate templates for go scripts, list existing commands and print paths to project and source code to facilitate editing and maintenance. 

## Philosophy

Scripting in Go should **_feel_** like scripting in other languages. You should be empowered to write scripts anywhere in the filesystem and expect them to work. You should be able to execute short scripts directly on the command line. You should be able to use a shebang to turn a go source file into an executable script. And unlike most scripting languages, it should be easy to keep track of your scripts and, if you want to, to use them globally like other system commands (e.g. cat, echo, ls, grep, find, etc.). There should be support to make writing short bits easier, like handling boilerplate and automating imports. Finally, it should be flexible. If you want to write a shebang script for regular use in a specific location, great. If you want to write a short throw-away one-liner with hard-coded arguments, great. If you want to write a reusable global command, great. It's your choice.    

## How It Works

The **goscript** executable will wrap any code specified on the command line with a main function and apply any required imports before compiling and optionally executing the code. If no name is given, the binary will be `[project folder]/bin/gocmd` and the source file will be `[project folder]/src/gocmd.go`. If the name is given, the binary and source files will reflect that name. By adding the `[project]/bin` folder to your PATH environment variable, the resulting binaries will be immediately available to execute like other system commands (such as ls, cat, echo, grep, etc.). 

If the --file option is used, then **goscript** will assume the file is a complete go source file and build it **_as is_**, rather than attempting to add imports and wrap code in a main function. However, to facilitate writing the go source file, the --template option is provided to print out a skeleton go source file as a starting point. That template can include imports and some basic code to start from if the --code option is provided. If the --name option is provided, the template will be saved to the project `src` folder for better IDE support when editing. The --edit option will open the file in the project src folder using your chosen editor. 

See examples, below, for more details. 

## Install

**Goscript** is ultimately about compiling Go code into binaries, so a Go module project is required to build your scripts. Further, the resulting binaries need to be on your path to be accessible. Follow these steps to get setup:

### Option 1 - Clone or Fork This Repo

1. Clone this repo and re-build the binary (if necessary ... binary was compiled on linux) with `go build -o goscript main.go`

2. Edit your **PATH** environment variable to include:
   1. **The project directory** created in Step 1 (ie. the location of the **goscript** binary). This will enable the **goscript** command to be executed from anywhere in your filesystem.
   2. **The `bin` sub-directory within the project directory** (ie. the location where all of the binaries for your scripts will be written). This will enable your scripted commands to be executed immediately from anywhere in your filesystem.
   
3. Optionally set the GOSCRIPT_EDITOR (or EDITOR) environment variable to the name of the editor you prefer to use for editing (e.g. "code" or "vim").

### Option 2 - Go Install and goscript --setup

1. Call `go install github.com/fkmiec/goscript@latest` This will install the goscript binary, which should ensure it is on your **PATH**. 

2. Call `goscript --setup <project name>` to setup a new project to host go scripts and follow instructions to set required environment variables.
   1. Set environment variable **GOSCRIPT_PROJECT_DIR** to the directory of your new project. 
   2. Add **$GOSCRIPT_PROJECT_DIR/bin** to the **PATH** environment variable
   
3. Optionally set the GOSCRIPT_EDITOR (or EDITOR) environment variable to the name of the editor you prefer to use for editing (e.g. "code" or "vim").

## Usage
```
Usage: goscript [options]
Options:
  --code|-c string
	    The code of your command or the name of a file containing the body of the main function.
  --file|-f string
	    A go src file, complete with main function and imports. Alternative to --code.
  --exec|-x
	    Execute the resulting binary.
  --name|-n string
	    A name for your command. The code will be saved to the project src directory with that name.
  --edit|-e string
	    Edit the named command in the editor specified by environment variable GOSCRIPT_EDITOR or EDITOR.
  --template|-t
	    Print a template go source file to stdout, or to the project src directory if --name provided.
  --list|-l
	    Print the list of existing commands.
  --path|-p string
	    Print the path to the source file specified, if exists in the project. Blank if not found.
  --cat string
  	  Prints the named script to stdout. The source and binary remain in the project.
  --export string
	    Exports the named script to stdout with shebang added and removes source and binary from project.
  --export-bin string
	    Exports the named binary to the local directory and removes source and binary from project.
  --delete string
	    Delete the specified compiled command. Removes .go extension from source file so it remains recoverable.
  --goget|-g string
	    Go get an external package (not part of stdlib) to pull into the project.
  --recompile
	    Recompile existing source files in the project src directory.
  --setup
	    A name, absolute path or 'help'. Creates a module project to be used by goscript. If 'help', prints setup instructions.
  --dir|-d
	    Print the directory path to the project.
  --bang|-b
	    Print the expected shebang line.
  --version|-v
	    Print the goscript version.

Example (Compile as 'hello'. Execute hello.):
  goscript --code 'script.Echo("Hello World!\n").Stdout()' --name hello; hello

Example (Execute immediately.):
  goscript --exec --code 'script.Echo("Hello World!\n").Stdout()'

Example shebang in 'myscript.go' file:
  (1) Add '#!/usr/bin/env -S goscript' to the top of your go source file.
  (2) Set execute permission and type "./myscript.go" as you would with a shell script.
```

## Examples

NOTE - For clarity, the long-form flags are used in the examples. 

### Compile and Execute in Two Steps

The code is compiled into a binary called `gocmd` in the `[project]/bin` folder. Since that folder is on your PATH, the command is immediately available to execute system-wide. 

```
> $ goscript --code 'script.FindFiles("/home/user/.config").Match("vlc").Stdout()'
> $ gocmd                                                                      
/home/user/.config/vlc/vlc-qt-interface.conf
/home/user/.config/vlc/vlcrc
```

### Compile and Execute in One Step with --exec

Adding the --exec option will cause the code to be executed immediately after compilation (similar to 'go run').

```
> $ goscript --exec --code 'script.FindFiles("/home/user/.config").Match("vlc").Stdout()'
/home/user/.config/vlc/vlc-qt-interface.conf
/home/user/.config/vlc/vlcrc
```

### Name the Command for Repeat Use

```
> $ goscript --name 'gofind' --code 'script.FindFiles("/home/user/.config").Match("vlc").Stdout()'
> $ gofind                                                                     
/home/user/.config/vlc/vlc-qt-interface.conf
/home/user/.config/vlc/vlcrc
```

### Required Imports Added Automatically 

If you need to pass command-line arguments, for instance, you might need to import the "os" package.  

```
> $ goscript --name 'gofind' --code 'script.FindFiles(os.Args[1]).Match(os.Args[2]).Stdout()'
> $ gofind '/home/user/.config' 'interface'                                  
/home/user/.config/vlc/vlc-qt-interface.conf
```  

**Goscript** examines the code and matches it to a map of package alias to package name covering the Go standard library (and "github/bitfield/script"). If code supplied using the --code option contains any of the pkg aliases defined in the map, goscript will automatically add the import to the generated source file. The intent is to reduce the amount of typing for short scripts entered using the --code option. The following example produces a template, illustrating the imports are added automatically.

```
> $ goscript --template --code 'fmt.Printf("ToPath: %s\n", path.Join(os.Args[1:]...))' one two three
#!/usr/bin/env -S goscript
package main

import ( 
    "fmt"
    "path"
    "os"
)

func main() {
    fmt.Printf("ToPath: %s\n", path.Join(os.Args[1:]...))
}
```

If the --exec option is provided, it produces the expected output:

```
> $ goscript --exec --code 'fmt.Printf("ToPath: %s\n", path.Join(os.Args[1:]...))' one two three
ToPath: one/two/three
```

**NOTE** - The built-in imports map can be augmented from an imports.json file in the project directory. If you require a third-party package, `goscript --goget [package name]` will add the package to the go.mod file as well as the imports.json file. You can also modify the pkg alias (ie. the key in the map) to allow you to use a shorter alias (e.g. "re" instead of "regexp"). 

This feature only applies to the --code option. It has no impact on code supplied through the --file option or in a shebang (see below) script.

### Optionally Use a File with --code

Go code won't always fit cleanly on the command line. You can still use the --code option to wrap code and add imports while pulling the body of the code from a file. This is a middle ground between putting everything on the command line and writing a full-fledged go source file with the --file option (see below). For example, if you have these contents in a file named "getip":

```
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
```

You can use the --code option to pull it in, wrap it in main(), add imports and execute. 

```
> $ goscript --exec --code getip
73.8.23.11
IPv4
US
United States
America/Chicago
7922
COMCAST-7922
73.8.0.0/15
```

### Use --file to Pass a Source File

Go scripts are ultimately just Go code. At minimum, that requires a main function, package declaration and imports. For short scripts passed with the --code option, **goscript** will help assemble a template Go source file. For more complex scripts read in using the --file option, **goscript** assumes you will provide a complete go source file. The file may include, for example, variables, structs and other functions besides main. The --template option can be specified to have **goscript** provide a boilerplate source file to start from. 

Once you have a source file, you can use the --file option to pass it to **goscript** to have it compiled and placed in the project and on the PATH so it is immediately executable. 

For example, if you have this source file named "gofind.go": 

```
package main

import ( 
    "os"
    "github.com/bitfield/script"
)

func main() {
    script.FindFiles(os.Args[1]).Match(os.Args[2]).Stdout()
}

```

You specify the path to the file using the --file option. If you supply the --name option, the source file will be saved in the project for future edits.  

```
> $ goscript --file '/tmp/gofind.go' --name 'findItNow'                        
> $ findItNow '/home/user/.config' 'interface'                               
/home/user/.config/vlc/vlc-qt-interface.conf
```

NOTE: If you pass the --name option, a **_copy_** of the source file is saved under that name in the project. The original file is not deleted or moved. Cleanup is at your discretion.

### Shebang (Linux and Mac only)

You can add a shebang (ie. #!/path/to/my/command) to a go source file to make it executable like a shell script.  

Just add `#!/usr/bin/env -S goscript` to the top of your go source file. The --template option includes the shebang in the template code it prints out. 

```
#!/usr/bin/env -S goscript

package main

import ( 
    "github.com/bitfield/script"
)

func main() {
    script.FindFiles("/home/user/.config").Match("interface").Stdout()
}
```

The source file does not need to have the .go file extension. 

Modify permissions to make it executable and run it directly like a shell script:

```
> $ ./gofind.go                                                               
/home/user/.config/vlc/vlc-qt-interface.conf

```

Running with shebang will be slightly less efficient since it will recompile the script each time it is executed. You may include the --name [name] option in the shebang line, or pass it as an additional argument on the command line the first time you execute the script (e.g. `./myscript --name mycommand`), in order to have the script compiled with a unique name. Thereafter, you can invoke the compiled script by that name (e.g. `mycommand`) for improved efficiency. 

### List Saved Commands

Can't remember that command you wrote last week? The --list option will show your previously-compiled commands.

```
> $ goscript --list 
gocmd
gofind
greet
shebang
```

### Use --edit Option to Edit a Command's Source in Context of the Project

For convenience, the --edit option takes the name of a command and will open the `[project]/src/[command].go` file in your preferred editor (specified by environment variable GOSCRIPT_EDITOR or EDITOR).

```
> $ goscript --edit gofind
``` 

If you use an IDE, such as VSCode, and are accustomed to that type of tool support when editing Go files, having the source file within the project (as opposed to a local .go file or shebang script) ensures that your editor has the project context it requires, including go.mod, go.sum and potentially a vendor folder. When done editing, execute `goscript -n [command]` to recompile.

NOTE - If the environment variables are not set, **Goscript** will output a helpful reminder to set them.

### Use --cat Option to Print a Command's Source to Stdout

The --cat option will print the source of a command to stdout. You can use this to quickly inspect the source or to write it to a local file as the basis for a new local script. Unlike the --export option (see below), the source and binary remain in the project. 

```
> $ goscript --cat gofind
``` 

### Use --export Option to Export a Command's Source and Remove the Command from the Project

The --export option writes the source of a command, with the shebang added at the top, to stdout. This is intended to facilitate converting a global command on the PATH into a local script. The function of the --delete option (see below) is invoked after the binary is moved. You can use --cat option if you simply want to see the source of a command or want to use it as a starting point for a new command or script. 

```
> $ goscript --export gofind
``` 

### Use --export-bin Option to Export a Command's Binary to the Current Directory and Remove it From the Project

The --export-bin option moves the binary for a command from the project to the current directory. This is intended to facilitate converting a global command on the PATH into a local command. The function of the --delete option (see below) is invoked after the binary is moved.  

```
> $ goscript --export-bin gofind
``` 

### Use --delete Option to "Soft Delete" a Command

With the --delete option, the binary for the command is deleted and the source for the command is renamed without the .go extension in the project src folder. This "soft delete" ensures the source code is preserved and can be recovered while it will be ignored by **Goscript** for all intents and purposes.

```
> $ goscript --delete gofind
``` 

### Get Path to Project (support project maintenance)

Need to clean up some old commands from the bin and src folders? Get the path to the project directory with the --dir option. 

```
> $ goscript --dir 
/home/user/go/src/github.com/fkmiec/goscript
```

### Get Path to Source File (support editing)

With the --name option, a copy of the source code is saved in the project src directory under that name. You can then use the --path option to print the path to the specified source file so that you can open it in your favorite editor and make updates (see also the --edit option). When done, calling goscript with just the --name option (without --code or --file) will cause the updated source file to be recompiled. Of course, you can navigate to the project folder and compile manually, but using **Goscript** helps to ensure consistency.

```
> $ goscript --name 'shebang' --file ./tmp.go 

> $ goscript --path 'shebang' 
/home/user/go/src/github.com/fkmiec/goscript/src/shebang.go

(file editing omitted from example)

> $ goscript --name 'shebang'                                                                                                                                

> $ shebang  
Hello Shebang!
``` 

### Recompile Existing Commands

For convenience, if you modify the sources in the project, or you clone your goscript repo to another machine with a different architecture, you can invoke `goscript --recompile` to recompile all existing commands. 

### Pipe Goscript Commands Together With Unix Commands

While this is primarily a function of the bitfield/scripts package, it's notable that you can combine your go scripts with existing Unix / Linux commands using pipes. 

```
> $ goscript --name 'uppercase' --code 'script.Stdin().FilterLine(strings.ToUpper).Stdout()'
                                                                                                                              
> $ echo 'hello world!' | uppercase 
HELLO WORLD!
                                                                                                                              
> $ echo 'hello world!' | uppercase | wc -c 
13
```
