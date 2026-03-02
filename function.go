package main

import (
	tea "charm.land/bubbletea/v2"
	"os/exec"
)

func (m *module) GotoFile(n int) {
	if n>len(m.entries)-1 || n<0{
		return
	}
	if n>m.cursor{
		m.offset=min(max(len(m.entries)-m.height,0), max(m.offset, n+GAP-m.height))
	} else if n< m.cursor{
		m.offset=max(0, min(m.offset, n-GAP))
	}
	m.cursor=n
}

func Open(path string) tea.Cmd{
	return func() tea.Msg{
		err:= exec.Command("xdg-open", path).Start()
		if err!=nil{return err}
		return nil
	}
}

func OpenEdit(path string) tea.Cmd {
	c:= exec.Command("vim", path)
	return tea.ExecProcess(c, func(err error) tea.Msg{
		if err!=nil{
			return err
		}
		return  editorMsg{}
	})
}
