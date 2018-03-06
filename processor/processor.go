package processor

import (
	"github.com/monochromegane/go-gitignore"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

var ExtensionToLanguage = map[string]string{"scm": "Scheme", "asmx": "ASP.NET", "handlebars": "Handlebars", "less": "LESS", "csproj": "MSBuild", "hamlet": "Hamlet", "lds": "LD Script", "tcl": "TCL", "gd": "GDScript", "lsp": "Lisp", "go": "Go", "sv": "SystemVerilog", "xml": "XML", "ads": "Ada", "clj": "Clojure", "elm": "Elm", "ihex": "Intel HEX", "ede": "Emacs Dev Env", "ts": "TypeScript", "yaml": "YAML", "sml": "Standard ML (SML)", "adb": "Ada", "tsx": "TypeScript", "vbproj": "MSBuild", "pcc": "C++", "dockerfile": "Dockerfile", "yml": "YAML", "ly": "Happy", "coffee": "CoffeeScript", "rx": "Forth", "vert": "GLSL", "psl": "PSL Assertion", "md": "Markdown", "rake": "Rakefile", "ahk": "AutoHotKey", "dockerignore": "Dockerfile", "dart": "Dart", "bat": "Batch", "d": "D", "lisp": "Lisp", "h": "C Header", "cmd": "Batch", "abap": "ABAP", "c++": "C++", "p": "Prolog", "ml": "OCaml", "x": "Alex", "sitemap": "ASP.NET", "el": "Emacs Lisp", "cfm": "ColdFusion", "htm": "HTML", "cfc": "ColdFusion CFScript", "ec": "C", "tex": "TeX", "ex": "Elixir", "btm": "Batch", "ur": "Ur/Web", "targets": "MSBuild", "webinfo": "ASP.NET", "for": "FORTRAN Legacy", "scala": "Scala", "lean": "Lean", "lua": "Lua", "ceylon": "Ceylon", "gvy": "Groovy", "geom": "GLSL", "rb": "Ruby", "props": "MSBuild", "fth": "Forth", "forth": "Forth", "ftn": "FORTRAN Legacy", "asp": "ASP", "fish": "Fish", "irunargs": "Verilog Args File", "comp": "GLSL", "vala": "Vala", "wl": "Wolfram", "asax": "ASP.NET", "jl": "Julia", "erl": "Erlang", "asa": "ASP", "cxx": "C++", "org": "Org", "js": "JavaScript", "swift": "Swift", "bash": "BASH", "c": "C", "lucius": "Lucius", "xrunargs": "Verilog Args File", "upkg": "Unreal Script", "idr": "Idris", "nix": "Nix", "cogent": "Cogent", "cob": "COBOL", "toml": "TOML", "oz": "Oz", "srt": "SRecode Template", "pde": "Processing", "hbs": "Handlebars", "mad": "Madlang", "markdown": "Markdown", "cabal": "Cabal", "pgc": "C", "cc": "C++", "hex": "HEX", "sql": "SQL", "cpp": "C++", "vim": "Vim Script", "ipp": "C++ Header", "xtend": "Xtend", "e4": "Forth", "cs": "C#", "cr": "Crystal", "txt": "Plain Text", "uci": "Unreal Script", "ckt": "Spice Netlist", "java": "Java", "fsproj": "MSBuild", "hrl": "Erlang", "py": "Python", "makefile": "Makefile", "f83": "Forth", "gtpl": "Groovy", "json": "JSON", "master": "ASP.NET", "asm": "Assembly", "f03": "FORTRAN Modern", "frt": "Forth", "rhtml": "Ruby HTML", "f08": "FORTRAN Modern", "pl": "Perl", "pm": "Perl", "mm": "Objective C++", "hx": "Haxe", "ascx": "ASP.NET", "hs": "Haskell", "xaml": "XAML", "jai": "JAI", "pfo": "FORTRAN Legacy", "aspx": "ASP.NET", "hh": "C++ Header", "vue": "Vue", "fsx": "F#", "f95": "FORTRAN Modern", "rst": "ReStructuredText", "fsscript": "F#", "dtsi": "Device Tree", "groovy": "Groovy", "polly": "Polly", "thy": "Isabelle", "f": "FORTRAN Legacy", "julius": "Julius", "ada": "Ada", "mli": "OCaml", "mk": "Makefile", "kts": "Kotlin", "r": "R", "svg": "SVG", "dts": "Device Tree", "vhd": "VHDL", "qcl": "QCL", "zsh": "Zsh", "sty": "TeX", "def": "Module-Definition", "qml": "QML", "vb": "Visual Basic", "v": "Coq", "uc": "Unreal Script", "vg": "Verilog", "vh": "Verilog", "cassius": "Cassius", "pro": "Prolog", "cbl": "COBOL", "ccp": "COBOL", "inl": "C++ Header", "as": "ActionScript", "cobol": "COBOL", "svh": "SystemVerilog", "in": "Autoconf", "purs": "PureScript", "grt": "Groovy", "pas": "Pascal", "cmake": "CMake", "csh": "C Shell", "proto": "Protocol Buffers", "fpm": "Forth", "tese": "GLSL", "nb": "Wolfram", "hpp": "C++ Header", "s": "Assembly", "tesc": "GLSL", "html": "HTML", "pad": "Ada", "fst": "F*", "hxx": "C++ Header", "text": "Plain Text", "css": "CSS", "frag": "GLSL", "fr": "Forth", "fs": "F#", "ft": "Forth", "nim": "Nim", "urs": "Ur/Web", "y": "Happy", "fb": "Forth", "4th": "Forth", "hlean": "Lean", "cshtml": "Razor", "cljs": "ClojureScript", "mak": "Makefile", "php": "PHP", "jsx": "JSX", "agda": "Agda", "lidr": "Idris", "scss": "Sass", "e": "Specman e", "sass": "Sass", "ss": "Scheme", "fsi": "F#", "rs": "Rust", "m": "Objective C", "f90": "FORTRAN Modern", "cpy": "COBOL", "sh": "Shell", "urp": "Ur/Web Project", "exs": "Elixir", "kt": "Kotlin", "sc": "Scala", "f77": "FORTRAN Legacy", "mustache": "Mustache"}

/// Get all the files that exist in the directory
func walkDirectory(root string, output *chan *FileJob) {
	gitignore, gitignoreerror := gitignore.NewGitIgnore(filepath.Join(root, ".gitignore"))

	filepath.Walk(root, func(root string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		// Need to exclude git, hg, svn etc...
		if strings.HasPrefix(root, ".git/") {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			if gitignoreerror != nil || !gitignore.Match(filepath.Join(root, info.Name()), false) {

				extension := strings.ToLower(path.Ext(info.Name()))

				// if name starts with . don't trim UNLESS there is more than one
				if !strings.HasPrefix(info.Name(), ".") || strings.Count(info.Name(), ".") != 1 {
					extension = strings.TrimLeft(extension, ".")
				}

				if extension == "" {
					extension = strings.ToLower(info.Name())
				}

				language, ok := ExtensionToLanguage[extension]

				if ok {
					*output <- &FileJob{Location: root, Filename: info.Name(), Extension: extension, Language: language}
				}
			}
		}

		return nil
	})

	close(*output)
}

func Process() {
	// A buffered channel that we can send work requests on.
	fileReadJobQueue := make(chan *FileJob, runtime.NumCPU()*20)
	fileProcessJobQueue := make(chan *FileJob, runtime.NumCPU())
	fileSummaryJobQueue := make(chan *FileJob, runtime.NumCPU()*20)

	go walkDirectory(os.Args[1], &fileReadJobQueue)
	go fileReaderWorker(&fileReadJobQueue, &fileProcessJobQueue)
	go fileProcessorWorker(&fileProcessJobQueue, &fileSummaryJobQueue)
	fileSummerize(&fileSummaryJobQueue) // Bring it all back to you
}
