package completion

import (
	"fmt"
	"os"
	"path"
	"strings"
)

func prog() string {
	p, err := os.Executable()
	if err != nil || strings.HasSuffix(p, "main") {
		return "launchpad"
	}
	return path.Base(p)
}

// BashTemplate returns a completion script for bash
func BashTemplate() string {
	return fmt.Sprintf(`#! /bin/bash

_launchpad_bash_autocomplete() {
  if [[ "${COMP_WORDS[0]}" != "source" ]]; then
    local cur opts base
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    if [[ "$cur" == "-"* ]]; then
      opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} ${cur} --generate-bash-completion )
    else
      opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} --generate-bash-completion )
    fi
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
  fi
}

complete -o bashdefault -o default -o nospace -F _cli_bash_autocomplete %s
`, prog())
}

// ZshTemplate returns a completion script for zsh
func ZshTemplate() string {
	p := prog()
	return fmt.Sprintf(`#compdef %s

_launchpad_zsh_autocomplete() {
  local -a opts
  local cur
  cur=${words[-1]}
  if [[ "$cur" == "-"* ]]; then
    opts=("${(@f)$(_CLI_ZSH_AUTOCOMPLETE_HACK=1 ${words[@]:0:#words[@]-1} ${cur} --generate-bash-completion)}")
  else
    opts=("${(@f)$(_CLI_ZSH_AUTOCOMPLETE_HACK=1 ${words[@]:0:#words[@]-1} --generate-bash-completion)}")
  fi

  if [[ "${opts[1]}" != "" ]]; then
    _describe 'values' opts
  else
    _files
  fi

  return
}

compdef _launchpad_zsh_autocomplete %s
`, p, p)
}
