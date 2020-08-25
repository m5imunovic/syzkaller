# Report generation

Documentation provided here is valid for linux/amd64 target architecture.

# Reading symbols
In order to read symbols from the _vmlinux_ binary syzkaller utilizes _nm_:
```
nm -Ptx vmlinux
```
flag -P means portable output format (Standard Output)
flat -t writes each numeric value in the specified format
        dependent on the single char used as the format option.
        Here x is used, meaning that the offset is written in hexadecimal.

Output is of the following form:
```
tracepoint_module_nb d ffffffff84509580 0000000000000018
...
udp_lib_hash t ffffffff831a4660 0000000000000007
```

First column is symbol name, second column its type (e.g. text section,
data section, debugging symbol, undefined, zero-init section, etc.), third
is symbol value in hex foramt, forth column is the size of an object
The final output is a list of symbols (pair with symbol address start
and address end). This is the task of readSymbols function in 
pkg/cover/report.go

Finally readSymbols function collects the starting and the ending
pair address of symbols.

## Object Dump and Symbolize

Dissasembling Linux executable binary and determining the frames in the
code is the task of objdumpAndSymbolize pkg/cover/report.go.

### Disassembling vmlinux

/usr/bin/objdump -d --no-show-raw-insn vmlinux
-d  means disassemble executable code blocks
-no-show-raw-insn prevents printing hex alongside symbolic disassembly

Example output looks like this:
```
ffffffff81000019:	pop    %rsi
ffffffff8100001a:	add    $0x51a6000,%rax
ffffffff81000020:	jmp    ffffffff81000042 <secondary_startup_64+0x12>
ffffffff81000022:	nopl   0x0(%rax)
ffffffff81000026:	nopw   %cs:0x0(%rax,%rax,1)
```

From this output coverage trace calls are identified to determine the 
start of the executable block addresses:
```
ffffffff8148ba08:	callq  ffffffff81382a00 <__sanitizer_cov_trace_pc>
```

### Translating addresses
```
addr2line -afi -e vmlinux
```

-afi means show addresses, function names and unwind inlined functions
-e is switch for specifying executable (vmlinux) instead of using default

addr2line reads hexadecimal addresses from standard input and prints the filename
function and line number for each address on standard output. Example usage:
```
>> ffffffff8148ba08
<< 0xffffffff8148ba08
<< generic_file_read_iter
<< /home/user/linux/mm/filemap.c:2363
```

This is the main task of the pkg/symbolizer/symbolizer.go

The final goal is to have a hash table of frames where key is a program counter
and value is a Frame consisting of a following information:
- PC - 64bit program counter value (same as key)
- Func - function name
- File - file where function code is located
- Line - Line in file where function is located
- Inline - boolean inlining information

Program counters collected during the syzkaller run are read from file 
(they can be exported using /rawcover http request handler


## Creating report

Agregate program counters, symbols and frames for for all the programs provided to 
ReportGenerator. Covered Pcs have frame information available. For those frames
collect the information about files and lines that are covered in the programs.

