package grok

import (
	"fmt"
)

import (
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
)

var Command = cmd.Cmd(
	"grok",
	`[options] <pkg>`,
	`
Option Flags
    -h,--help                         Show this message
`,
	"",
	[]string{},
	func(args []string, optargs []getopt.OptArg, xtra ...interface{}) ([]string, interface{}, *cmd.Error) {
		// for _, oa := range optargs {
		// 	switch oa.Opt() {
		// 	}
		// }
		fmt.Println("grokked", args, optargs, xtra)
		return args, "grok", nil
	},
)

