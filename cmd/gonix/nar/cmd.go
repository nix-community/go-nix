package nar

type Cmd struct {
	Cat      CatCmd      `kong:"cmd,name='cat',help='Print the contents of a file inside a NAR file'"`
	DumpPath DumpPathCmd `kong:"cmd,name='dump-path',help='Serialise a path to stdout in NAR format'"`
	Ls       LsCmd       `kong:"cmd,name='ls',help='Show information about a path inside a NAR file'"`
}
