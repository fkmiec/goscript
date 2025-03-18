# goscripts

**Goscripts** is a utility to make it easier to use Go as a scripting language. It was inspired by and implemented to leverage https://github.com/bitfield/script, a Go package aimed at bringing unix-like piped commands to Go.

The Go compiler is fast enough that using Go for scripting tasks can be a good choice. There are challenges, however. Modern Go requires modules and does not support "go get -u" and the concept of a global GOPATH. Other projects have attempted to do something similar, but either fail to account for some of these challenges or create and delete temporary projects, folders and files for every execution and don't provide an efficient or effective way to re-use your scripts. 

Enter **Goscripts**.  

## Features

* Execute simple go code directly on the command line
* Write go source files and execute them like a script with shebang
* Compile go code into named binaries for repeated use. Build up a library of custom commands. 
* Everything in one project, accessible system-wide through the **goscript** command. 
* Options to generate boilerplate for go scripts, list previously compiled commands and print paths to project and source code to enable editing and maintenance. 

## How It Works

The **goscript** executable will wrap any code specified on the command line with a main function and apply any specified imports before compiling and optionally executing the code. If no name is given, the binary will be `[project folder]/bin/gocmd` and the source file will be `[project folder]/src/tmp.go`. If the name is given, the binary and source files will reflect that name. By adding the `[project]` and `[project]/bin` folders to your PATH environment variable, the resulting binaries will be immediately available to execute like other system commands (such as ls, cat, echo, grep, etc.). 

If the --file option is used, then **goscript** will assume the file is a complete go source file and build it _**as is**_, rather than attempting to add imports and wrap code in a main function. However, to facilitate writing the go source file, the --template option is provided to print out the typical boilerplate as a starting point. That template can be augmented with imports and some basic code to start from if the --imports and --code options are provided. 

See examples, below, for more details. 

## Install

**Goscripts** is ultimately about compiling Go code into binaries, so a Go module project is required to build your scripts. Further, the resulting binaries need to be on your path to be accessible. Follow these steps to get setup:

1. Clone this repo and build the binary with `go build -o goscript main.go`

2. Edit your PATH environment variable to include the project directory created in Step 1 (ie. the location of the **goscript** binary). This will enable the **goscript** command to be executed from anywhere in your filesystem.

3. Edit your PATH environment variable to include the `bin` sub-directory within the project directory (ie. the location where all of the binaries for your scripts will be written). This will enable your scripted commands to be executed immediately from anywhere in your filesystem.

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
  --save|-s
	    Save the source file <name>.go to the project src folder.
  --list|-l
	    Print the list of previously-compiled commands.
  --path|-p
	    Print the path to the source file, if name provided, or to the project.
  --template|-t
	    Print a template go source file to stdout. After edits, use --file to compile with goscript.
  --exec|-x
	    Execute the resulting binary.

Example (Compile with default name gocmd. Execute gocmd.):
  goscript --code "script.Echo(\" Hello World! \").Stdout()";gocmd

Example shebang in 'myscript.go' file:
  (1) Add '#!/usr/bin/env -S goscript -x -f' to the top of your go source file.
  (2) Set execute permission and type "./myscript.go" as you would with a shell script.
```

## Examples

### Compile and Execute in Two Steps (Recommended)

With the --code (or -c) option, the code is compiled into a binary called `gocmd` in the `[project]/bin` folder.
```
> $ goscript -c 'script.FindFiles("/home/fkmiec/.config").Match("vlc").Stdout()'
> $ gocmd                                                                      
/home/fkmiec/.config/vlc/vlc-qt-interface.conf
/home/fkmiec/.config/vlc/vlcrc
```

### Compile and Execute in One Step with --exec

Adding the --exec option will cause the code to be executed immediately after compilation (similar to 'go run').

```
> $ goscript --exec -c 'script.FindFiles("/home/fkmiec/.config").Match("vlc").Stdout()'
/home/fkmiec/.config/vlc/vlc-qt-interface.conf
/home/fkmiec/.config/vlc/vlcrc
```

NOTE - While it may seem convenient or efficient to use --exec (or -x) to execute immediately, there are limitations with asking goscript to invoke the execution, rather than compiling first and executing the command directly. For one, when using --exec, there is no good way to pass arguments or pipe to stdin since anything passed (or piped) will be assumed to be inputs to **goscript**, rather than the resulting binary command. Accordingly, it's recommended to compile first and separately invoke the resulting binary. 

If desired, you can put both commands on the same line in the terminal, separated by a semi-colon. In the example below, which omits the --exec option, the unnamed script will use the default name of 'gocmd', which we add with a semi-colon. The code appears to execute in one step. 

```
> $ goscript -c 'script.FindFiles("/home/fkmiec/.config").Match("vlc").Stdout()';gocmd
/home/fkmiec/.config/vlc/vlc-qt-interface.conf
/home/fkmiec/.config/vlc/vlcrc
```

### Name the Command for Repeat Use

```
> $ goscript --name 'gofind' -c 'script.FindFiles("/home/fkmiec/.config").Match("vlc").Stdout()'
> $ gofind                                                                     
/home/fkmiec/.config/vlc/vlc-qt-interface.conf
/home/fkmiec/.config/vlc/vlcrc
```

### Use --imports to Leverage Additional Packages

If you need to pass command-line arguments, for instance, you need to import the "os" package.

```
> $ goscript --name 'gofind' --imports 'os' -c 'script.FindFiles(os.Args[1]).Match(os.Args[2]).Stdout()'
> $ gofind '/home/fkmiec/.config' 'interface'                                  
/home/fkmiec/.config/vlc/vlc-qt-interface.conf
```

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

You can specify the path to the file using the --file option. If you supply --name and --save options, the source file will be saved in the project for future edits.  

```
> $ goscript --file '/tmp/gofind.go' --name 'findItNow' --save                        
> $ findItNow '/home/fkmiec/.config' 'interface'                               
/home/fkmiec/.config/vlc/vlc-qt-interface.conf
```

NOTE: If you pass the --save option, a copy of the source file is saved in the project. The original file is not deleted or moved. Cleanup is at your discretion.

### Shebang

Using a shebang (ie. #!/path/to/my/command) to make the source file executable like a shell script sounds really appealing and is supported, with some limitations. 

Just add `#!/usr/bin/env -S goscript -x -f` to the top of your go source file. 

```
#!/usr/bin/env -S goscript -x -f

package main

import ( 
    "github.com/bitfield/script"
)

func main() {
    script.FindFiles("/home/fkmiec/.config").Match("interface").Stdout()
}
```

Modify permissions to make it executable and run it directly like a shell script:

```
> $ ./gofind.go                                                               
/home/fkmiec/.config/vlc/vlc-qt-interface.conf

```

In terms of limitations, with shebang the script has to be recompiled every time and you will run up against the same inability to pass arguments or pipe input to it that was mentioned in the --exec example above.

### List Saved Commands

Can't remember that command you wrote last week? The --list option will show your previously-compiled commands.

```
> $ goscript --list 
Commands:
	gocmd
	gofind
	greet
	shebang
```

### Get Path to Project (support project maintenance)

Need to clean up some old commands from the bin and src folders? Get the path to the project with the --path option. 

```
> $ goscript --path 
/home/fkmiec/go/src/github.com/fkmiec/goscripts
```

### Get Path to Source File (support editing)

With the --save option, you can save a copy of the source code for your named command. You can then use the --path option together with --name to print the path to that specific source file so that you can open it in your favorite editor and make updates. When done, calling goscripts with the just the --name option (without --code or --file) will cause the updated source file to be recompiled. Of course, you can navigate to the project folder and compile manually, but using **goscript** helps to ensure consistency.

```
> $ goscript --name 'shebang' --save --file ./tmp.go 

> $ goscript --path --name 'shebang' 
/home/fkmiec/go/src/github.com/fkmiec/goscripts/src/shebang.go

(file editing omitted from example)

> $ goscript --name 'shebang'                                                                                                                                

> $ shebang  
Hello Shebang!

```

### Pipe Goscripts Commands Together With Unix Commands

While this is primarily a function of the bitfield/scripts package, it's notable that you can combine your go scripts with existing Unix / Linux commands using pipes. 

```
> $ goscript --name 'uppercase' --save --imports 'strings' --code 'script.Stdin().FilterLine(strings.ToUpper).Stdout()'
                                                                                                                              
> $ echo 'hello world!' | uppercase 
HELLO WORLD!
                                                                                                                              
> $ echo 'hello world!' | uppercase | wc -c 
13
```
