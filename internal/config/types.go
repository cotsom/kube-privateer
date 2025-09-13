package config

type CmdResult struct {
	Cmd    []string `json:"cmd"`
	Stdout string   `json:"stdout"`
	Stderr string   `json:"stderr"`
	Error  string   `json:"error,omitempty"`
}
