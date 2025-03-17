# goscripts

Code to facilitate using go-script library to write unix-like scripts in go. 

The goscript executable will wrap any code specified on the command line in a main function and apply any specified imports before compiling and optionally running the code. If no name is given, the binary will be "<project>/bin/gocmd" and the source file will be "<project>/src/tmp.go". If the name is given, the binary and source files will reflect that name. By adding the "<project>" and "<project>/bin" folders to your PATH environment variable, the resulting binaries will be immediately available to execute like any other system commands. 

If a file is specified, then goscript will assume it is a complete go source file and build it as is, rather than attempting to add imports and wrap code in a main function. This is reasonable since that bit of boilerplate is easy to write and there are too many potential errors trying to determine if imports were given, variables were defined, and so on, and then insert them where needed. However, to facilitate writing the go source file, the -template option is provided to print out the typical boilerplate as a starting point. That template can be augmented with imports and some basic code to start from if the -imports and -code options are provided. 

A similar option for a template file to start from would be to specify the -savesource and -name options to cause a named source file to be created in the project src folder. You can then get the path to that source file and edit it in your favorite editor. If you then provide the -name without the -code or -file options, it will pull the source file from the src folder and recompile it. 
