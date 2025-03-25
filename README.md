# goscript

**Goscript** is a utility to make it easier to use Go as a scripting language. It was inspired by and implemented to leverage https://github.com/bitfield/script, a Go package aimed at bringing unix-like piped commands to Go.

The Go compiler is fast enough that using Go for scripting tasks can be a good choice. There are challenges, however. Modern Go requires modules and does not support "go get -u" and the concept of a global GOPATH. Other projects have attempted to do something similar, but either fail to account for some of these challenges or create and delete temporary projects, folders and files for every execution and don't provide an efficient or effective way to re-use your scripts. 

Enter **Goscript**.  

- [goscript](#goscript)
  - [Features](#features)
  - [How It Works](#how-it-works)
  - [Install](#install)
  - [Usage](#usage)
  - [Examples](#examples)
    - [Compile and Execute in Two Steps](#compile-and-execute-in-two-steps)
    - [Compile and Execute in One Step with --exec](#compile-and-execute-in-one-step-with---exec)
    - [Name the Command for Repeat Use](#name-the-command-for-repeat-use)
    - [Use --imports to Leverage Additional Packages](#use---imports-to-leverage-additional-packages)
    - [Use --file to Pass a Source File](#use---file-to-pass-a-source-file)
    - [Shebang](#shebang)
    - [List Saved Commands](#list-saved-commands)
    - [Get Path to Project (support project maintenance)](#get-path-to-project-support-project-maintenance)
    - [Get Path to Source File (support editing)](#get-path-to-source-file-support-editing)
    - [Recompile Existing Commands](#recompile-existing-commands)
    - [Pipe Goscript Commands Together With Unix Commands](#pipe-goscript-commands-together-with-unix-commands)

## Features

* Execute simple go code directly on the command line
* Write go source files and execute them like a script with shebang
* Compile go code into named binaries for repeated use. Build up a library of custom commands. 
* Everything in one project, accessible system-wide through the **goscript** command. 
* Options to generate boilerplate for go scripts, list previously compiled commands and print paths to project and source code to facilitate editing and maintenance. 

## How It Works

The **goscript** executable will wrap any code specified on the command line with a main function and apply any specified imports before compiling and optionally executing the code. If no name is given, the binary will be `[project folder]/bin/gocmd` and the source file will be `[project folder]/src/gocmd.go`. If the name is given, the binary and source files will reflect that name. By adding the `[project]` and `[project]/bin` folders to your PATH environment variable, the resulting binaries will be immediately available to execute like other system commands (such as ls, cat, echo, grep, etc.). 

If the --file option is used, then **goscript** will assume the file is a complete go source file and build it _**as is**_, rather than attempting to add imports and wrap code in a main function. However, to facilitate writing the go source file, the --template option is provided to print out the typical boilerplate as a starting point. That template can include imports and some basic code to start from if the --imports and --code options are provided. 

See examples, below, for more details. 

## Install

**Goscript** is ultimately about compiling Go code into binaries, so a Go module project is required to build your scripts. Further, the resulting binaries need to be on your path to be accessible. Follow these steps to get setup:

1. Clone this repo and re-build the binary (if necessary ... binary was compiled on linux) with `go build -o goscript main.go`

2. Edit your PATH environment variable to include:
   1. The project directory created in Step 1 (ie. the location of the **goscript** binary). This will enable the **goscript** command to be executed from anywhere in your filesystem.
   2. The `bin` sub-directory within the project directory (ie. the location where all of the binaries for your scripts will be written). This will enable your scripted commands to be executed immediately from anywhere in your filesystem.

NOTE: If you prefer to create another Go project for your scripts, rather than using the clone of this repo, you can do so. Just remember to (1) copy the **goscript** binary to the root of that new project folder, (2) create "src" and "bin" sub-folders expected by **goscript**, and (3) after running `go mod init (project name)` to initiate a module project, call `go get github.com/bitfield/script`, which is expected by **goscript**. Adjust the PATH instructions, above, for your new project's path. 

## Usage
```
Usage: goscript [options]
Options:
  --code|-c string
	    The code of your command. Defaults to empty string.
  --imports|-i string
	    A comma-separated list of go packages to import. Not used with --file option.
  --file|-f string
	    A go src file, complete with main function and imports. Alternative to --code and --imports options.
  --name|-n string
	    A name for your command. Defaults to gocmd.
  --list|-l
	    Print the list of previously-compiled commands.
  --recompile
	    Recompile existing source files in the project src directory.
  --dir|-d
	    Print the directory path to the project.
  --path|-p
	    Print the path to the source file specified, if exists in the project. Blank if not found.
  --bang|-b
	    Print the expected shebang line.
  --template|-t
	    Print a template go source file to stdout. After edits, use --file to compile with goscript.
  --exec|-x
	    Execute the resulting binary.

Example (Compile as 'hello'. Execute hello.):
  goscript --code 'script.Echo(" Hello World!\n").Stdout()' --name hello; hello

Example (Execute immediately.):
  goscript --exec --code 'script.Echo(" Hello World!\n").Stdout()'

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

And alternative to passing the --exec flag would be to put both commands on the same line in the terminal, separated by a semi-colon. In the example below, which omits the --exec option, the unnamed script will use the default name of 'gocmd', which we add with a semi-colon. The code appears to execute in one step. 

```
> $ goscript --code 'script.FindFiles("/home/user/.config").Match("vlc").Stdout()';gocmd
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

### Use --imports to Leverage Additional Packages

If you need to pass command-line arguments, for instance, you might need to import the "os" package. 

```
> $ goscript --name 'gofind' --imports 'os' --code 'script.FindFiles(os.Args[1]).Match(os.Args[2]).Stdout()'
> $ gofind '/home/user/.config' 'interface'                                  
/home/user/.config/vlc/vlc-qt-interface.conf
```

**NOTE** - There is a util/imports.go file in the project that defines a map of pkg alias to full pkg name for "github/bitfield/script" and the go stdlib. If code supplied using the --code option contains any of the pkg aliases defined in the map, goscript will automatically add the import to the generated source file. The intent is to reduce the amount of typing for short scripts entered directly on the command-line. The following example produces a template, illustrating the imports are added automatically. 

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

You can add other packages to the util/Imports file and recompile for those you use frequently. You can also modify the pkg alias (ie. the key in the map) to allow you to use a shorter alias (e.g. "re" instead of "regexp"). 

This feature has no impact on code supplied in files (either through --file or using a shebang in a script).  

### Use --file to Pass a Source File

Go scripts are ultimately just go code. At minimum, that requires a main function, package declaration and imports. For simple one-liners, **goscript** will help assemble a boilerplate go source file. For more complex scripts, **goscript** assumes you will provide a complete go source file. The file may include, for example, variables and other functions besides main. The --template option can be specified to have **goscript** provide a boilerplate source file to start from. 

Once you have a source file, you can use the --file option to pass it to **goscript** and have it compiled and placed in the project and on the PATH so it is immediately executable. 

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

You can specify the path to the file using the --file option. If you supply the --name option, the source file will be saved in the project for future edits.  

```
> $ goscript --file '/tmp/gofind.go' --name 'findItNow'                        
> $ findItNow '/home/user/.config' 'interface'                               
/home/user/.config/vlc/vlc-qt-interface.conf
```

NOTE: If you pass the --name option, a copy of the source file is saved under that name in the project. The original file is not deleted or moved. Cleanup is at your discretion.

### Shebang

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

### Get Path to Project (support project maintenance)

Need to clean up some old commands from the bin and src folders? Get the path to the project directory with the --dir option. 

```
> $ goscript --dir 
/home/user/go/src/github.com/fkmiec/goscript
```

### Get Path to Source File (support editing)

With the --save option, you can save a copy of the source code for your named command. You can then use the --path option to print the path to the specified source file so that you can open it in your favorite editor and make updates. When done, calling goscript with the just the --name option (without --code or --file) will cause the updated source file to be recompiled. Of course, you can navigate to the project folder and compile manually, but using **goscript** helps to ensure consistency.

```
> $ goscript --name 'shebang' --save --file ./tmp.go 

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
> $ goscript --name 'uppercase' --save --imports 'strings' --code 'script.Stdin().FilterLine(strings.ToUpper).Stdout()'
                                                                                                                              
> $ echo 'hello world!' | uppercase 
HELLO WORLD!
                                                                                                                              
> $ echo 'hello world!' | uppercase | wc -c 
13
```
