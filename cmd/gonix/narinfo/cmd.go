package narinfo

type Cmd struct {
	Info InfoCmd `kong:"cmd,name='info',help='Show information about a narinfo file'"`
}
