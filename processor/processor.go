package processor

import (
	"fmt"
	"runtime"
)

// This is generated and set to be a map to be as fast a lookup as possible
var ExtensionToLanguage = map[string]string{"scm": "Scheme", "asmx": "ASP.NET", "handlebars": "Handlebars", "less": "LESS", "csproj": "MSBuild", "hamlet": "Hamlet", "lds": "LD Script", "tcl": "TCL", "gd": "GDScript", "lsp": "Lisp", "go": "Go", "sv": "SystemVerilog", "xml": "XML", "ads": "Ada", "clj": "Clojure", "elm": "Elm", "ihex": "Intel HEX", "ede": "Emacs Dev Env", "ts": "TypeScript", "yaml": "YAML", "sml": "Standard ML (SML)", "adb": "Ada", "tsx": "TypeScript", "vbproj": "MSBuild", "pcc": "C++", "dockerfile": "Dockerfile", "yml": "YAML", "ly": "Happy", "coffee": "CoffeeScript", "rx": "Forth", "vert": "GLSL", "psl": "PSL Assertion", "md": "Markdown", "rake": "Rakefile", "ahk": "AutoHotKey", "dockerignore": "Dockerfile", "dart": "Dart", "bat": "Batch", "d": "D", "lisp": "Lisp", "h": "C Header", "cmd": "Batch", "abap": "ABAP", "c++": "C++", "p": "Prolog", "ml": "OCaml", "x": "Alex", "sitemap": "ASP.NET", "el": "Emacs Lisp", "cfm": "ColdFusion", "htm": "HTML", "cfc": "ColdFusion CFScript", "ec": "C", "tex": "TeX", "ex": "Elixir", "btm": "Batch", "ur": "Ur/Web", "targets": "MSBuild", "webinfo": "ASP.NET", "for": "FORTRAN Legacy", "scala": "Scala", "lean": "Lean", "lua": "Lua", "ceylon": "Ceylon", "gvy": "Groovy", "geom": "GLSL", "rb": "Ruby", "props": "MSBuild", "fth": "Forth", "forth": "Forth", "ftn": "FORTRAN Legacy", "asp": "ASP", "fish": "Fish", "irunargs": "Verilog Args File", "comp": "GLSL", "vala": "Vala", "wl": "Wolfram", "asax": "ASP.NET", "jl": "Julia", "erl": "Erlang", "asa": "ASP", "cxx": "C++", "org": "Org", "js": "JavaScript", "swift": "Swift", "bash": "BASH", "c": "C", "lucius": "Lucius", "xrunargs": "Verilog Args File", "upkg": "Unreal Script", "idr": "Idris", "nix": "Nix", "cogent": "Cogent", "cob": "COBOL", "toml": "TOML", "oz": "Oz", "srt": "SRecode Template", "pde": "Processing", "hbs": "Handlebars", "mad": "Madlang", "markdown": "Markdown", "cabal": "Cabal", "pgc": "C", "cc": "C++", "hex": "HEX", "sql": "SQL", "cpp": "C++", "vim": "Vim Script", "ipp": "C++ Header", "xtend": "Xtend", "e4": "Forth", "cs": "C#", "cr": "Crystal", "txt": "Plain Text", "uci": "Unreal Script", "ckt": "Spice Netlist", "java": "Java", "fsproj": "MSBuild", "hrl": "Erlang", "py": "Python", "makefile": "Makefile", "f83": "Forth", "gtpl": "Groovy", "json": "JSON", "master": "ASP.NET", "asm": "Assembly", "f03": "FORTRAN Modern", "frt": "Forth", "rhtml": "Ruby HTML", "f08": "FORTRAN Modern", "pl": "Perl", "pm": "Perl", "mm": "Objective C++", "hx": "Haxe", "ascx": "ASP.NET", "hs": "Haskell", "xaml": "XAML", "jai": "JAI", "pfo": "FORTRAN Legacy", "aspx": "ASP.NET", "hh": "C++ Header", "vue": "Vue", "fsx": "F#", "f95": "FORTRAN Modern", "rst": "ReStructuredText", "fsscript": "F#", "dtsi": "Device Tree", "groovy": "Groovy", "polly": "Polly", "thy": "Isabelle", "f": "FORTRAN Legacy", "julius": "Julius", "ada": "Ada", "mli": "OCaml", "mk": "Makefile", "kts": "Kotlin", "r": "R", "svg": "SVG", "dts": "Device Tree", "vhd": "VHDL", "qcl": "QCL", "zsh": "Zsh", "sty": "TeX", "def": "Module-Definition", "qml": "QML", "vb": "Visual Basic", "v": "Coq", "uc": "Unreal Script", "vg": "Verilog", "vh": "Verilog", "cassius": "Cassius", "pro": "Prolog", "cbl": "COBOL", "ccp": "COBOL", "inl": "C++ Header", "as": "ActionScript", "cobol": "COBOL", "svh": "SystemVerilog", "in": "Autoconf", "purs": "PureScript", "grt": "Groovy", "pas": "Pascal", "cmake": "CMake", "csh": "C Shell", "proto": "Protocol Buffers", "fpm": "Forth", "tese": "GLSL", "nb": "Wolfram", "hpp": "C++ Header", "s": "Assembly", "tesc": "GLSL", "html": "HTML", "pad": "Ada", "fst": "F*", "hxx": "C++ Header", "text": "Plain Text", "css": "CSS", "frag": "GLSL", "fr": "Forth", "fs": "F#", "ft": "Forth", "nim": "Nim", "urs": "Ur/Web", "y": "Happy", "fb": "Forth", "4th": "Forth", "hlean": "Lean", "cshtml": "Razor", "cljs": "ClojureScript", "mak": "Makefile", "php": "PHP", "jsx": "JSX", "agda": "Agda", "lidr": "Idris", "scss": "Sass", "e": "Specman e", "sass": "Sass", "ss": "Scheme", "fsi": "F#", "rs": "Rust", "m": "Objective C", "f90": "FORTRAN Modern", "cpy": "COBOL", "sh": "Shell", "urp": "Ur/Web Project", "exs": "Elixir", "kt": "Kotlin", "sc": "Scala", "f77": "FORTRAN Legacy", "mustache": "Mustache"}

// Flags set via the CLI which control how the output is displayed
var Files = false
var Verbose = false
var Debug = false
var Trace = false
var SortBy = ""
var PathBlacklist = ""
var NoThreads = 0
var GarbageCollect = false

// Not set via flags but by arguments following the the flags
var DirFilePaths = []string{}

func Process() {
	// Clean up and invlid arguments before setting everything up
	if len(DirFilePaths) == 0 {
		DirFilePaths = append(DirFilePaths, ".")
	}

	// Less than or 0 threads is invalid so set to at least one
	if NoThreads <= 0 {
		// Need to consider if we should set https://golang.org/pkg/runtime/debug/#SetMaxThreads
		printDebug(fmt.Sprintf("NoThreads set to less than 0 changing to: %d", runtime.NumCPU()))

		// runtime.NumCPU() should never return zero or negative but lets be sure about that
		NoThreads = max(1, runtime.NumCPU())
	} else if NoThreads > 1000000 {
		// Anything over one million is likely to cause issues so lets limit
		printDebug(fmt.Sprintf("NoThreads set to greater than 1000000 changing to: 1000000"))
		NoThreads = 1000000
	}

	printDebug(fmt.Sprintf("NumCPU: %d", runtime.NumCPU()))
	printDebug(fmt.Sprintf("NoThreads: %d", NoThreads))
	printDebug(fmt.Sprintf("SortBy: %s", SortBy))
	printDebug(fmt.Sprintf("PathBlacklist: %s", PathBlacklist))

	fileListQueue := make(chan *FileJob, 1000)                      // Files ready to be read from disk
	fileReadJobQueue := make(chan *FileJob, min(100, NoThreads*10)) // Workers reading from disk
	fileReadContentJobQueue := make(chan *FileJob, 1000)            // Files ready to be processed
	fileProcessJobQueue := make(chan *FileJob, NoThreads)           // Workers doing the hard work
	fileSummaryJobQueue := make(chan *FileJob, 1000)                // Files ready to be summerised

	go walkDirectory(DirFilePaths[0], &fileListQueue)
	go fileBufferReader(&fileListQueue, &fileReadJobQueue)
	go fileReaderWorker(&fileReadJobQueue, &fileReadContentJobQueue)
	go fileBufferReader(&fileReadContentJobQueue, &fileProcessJobQueue)
	go fileProcessorWorker(&fileProcessJobQueue, &fileSummaryJobQueue)

	if Files {
		fileSummerizeFiles(&fileSummaryJobQueue)
	} else {
		fileSummerize(&fileSummaryJobQueue)
	}
}
