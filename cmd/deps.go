package cmd

import (
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	_ = cobra.Command{}
	_ = viper.GetViper
	_ = progressbar.New
)
