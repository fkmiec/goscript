# goscripts

**Goscripts** is a small utility to make it easier to use Go as a scripting language. It was inspired by and implemented to leverage https://github.com/bitfield/script, a Go package aimed at bringing unix-like piped commands to Go. 

The Go compiler is fast enough that using Go for scripting tasks can be a good choice. There are challenges, however, in that it must be compiled and that modern Go requires modules and does not support "go get -u" and the concept of a global GOPATH. Other solutions, such as goscript.sh, gosh and gorun, either fail to account for some of these challenges or create and delete temporary projects, folders and files repeatedly and don't really provide an efficient or effective way to re-use your scripts. 

Enter **Goscripts**.  

The **goscript** executable will wrap any code specified on the command line with a main function and apply any specified imports before compiling and optionally executing the code. If no name is given, the binary will be `[project folder]/bin/gocmd` and the source file will be `[project folder]/src/tmp.go`. If the name is given, the binary and source files will reflect that name. By adding the `[project]` and `[project]/bin` folders to your PATH environment variable, the resulting binaries will be immediately available to execute like any other system commands. 

If the --file option is used, then **goscript** will assume the file is a complete go source file and build it _**as is**_, rather than attempting to add imports and wrap code in a main function. It would be error-prone to attempt to anticipate and edit for the possible presence of existing imports, variables, functions and more in the file. However, to facilitate writing the go source file, the --template option is provided to print out the typical boilerplate as a starting point. That template can be augmented with imports and some basic code to start from if the --imports and --code options are provided. 

A similar option for a template file to start from would be to specify the --save and --name options to cause a named source file to be created in the project src folder. You can then get the --path to that source file and edit it in your favorite editor. If you then provide the --name without the --code or --file options, it will read the source file from the src folder and recompile it. 

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

### Compile and Execute in One Step with --exec

```
> $ goscript --exec -c 'script.FindFiles("/home/fkmiec/.config").Match("vlc").Stdout()'
/home/fkmiec/.config/vlc/vlc-qt-interface.conf
/home/fkmiec/.config/vlc/vlcrc
```

NOTE - While it may seem convenient or efficient to use --exec (or -x) to execute immediately, there are bound to be limitations with asking goscript to invoke the execution, rather than letting your shell do it. For one, when using --exec, there is no good way to pass arguments or pipe to stdin since anything passed (or piped) will be assumed to be inputs to **goscript**, rather than the resulting binary command. Accordingly, it's recommended to compile first and separately invoke the resulting binary. 

If desired, you can put both commands on the same line in the terminal, separated by a semi-colon. In the example below, the unnamed script will use the default name of 'gocmd', which we add with a semi-colon. The code appears to execute in one step. 

```
> $ goscript -c 'script.FindFiles("/home/fkmiec/.config").Match("vlc").Stdout()';gocmd
/home/fkmiec/.config/vlc/vlc-qt-interface.conf
/home/fkmiec/.config/vlc/vlcrc
```

### Compile and Execute in Two Steps

If we remove --exec, the code is compiled into a binary called `gocmd` in the `[project]/bin` folder.
```
> $ goscript -c 'script.FindFiles("/home/fkmiec/.config").Match("vlc").Stdout()'
> $ gocmd                                                                      
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

### Use --imports to Enable Passing Arguments

```
> $ goscript --name 'gofind' --imports 'os' -c 'script.FindFiles(os.Args[1]).Match(os.Args[2]).Stdout()'
> $ gofind '/home/fkmiec/.config' 'interface'                                  
/home/fkmiec/.config/vlc/vlc-qt-interface.conf
```

### Use --file to Pass a Source File

Go scripts are ultimately just go code. They require a main function and necessary imports. For simple one-liners, **goscript** will help assemble a boilerplate go source file. For more complex scripts, **goscript** assumes you will provide a complete go source file. The file may include, for example, variables and other functions besides main. The --template option can be specified to have **goscript** provide a boilerplace source file to start from. 

Once you have a source file, you can use the --file option to pass it to **goscript** and have it compiled and placed in the project and on the PATH so it is immediately executable. 

If you have this source file named "gofind.go": 

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

You can specify the path to the file using the --file option. 

```
> $ goscript --file '/tmp/gofind.go' --name 'findItNow'                        
> $ findItNow '/home/fkmiec/.config' 'interface'                               
/home/fkmiec/.config/vlc/vlc-qt-interface.conf
```

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

And run it directly like a shell script:

```
> $ ./gofind.go                                                               
/home/fkmiec/.config/vlc/vlc-qt-interface.conf

```

In terms of limitations, with shebang, the script has to be recompiled every time and you will run up against the same inability to pass arguments or pipe input to it that was mentioned in the first example above.

TBD

* Pipe from linux command to a go script to another linux command
* Get path to project (support editing)
* Get path to source file (support editing)