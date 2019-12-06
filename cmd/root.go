package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xnyo/papy/config"
)

// Verbose is true if and only if the -v flag is present
var Verbose bool

// Config is the global configuration file
var Config config.Configuration

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")
}

func initConfig() {
	// viper.SetDefault("compiler_path", "PapyrusCompiler.exe")
	viper.SetConfigName(".papy")
	viper.AddConfigPath("$HOME")
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			Fatal(fmt.Errorf("cannot read config file: %v", err))
		}
	}
	err := viper.Unmarshal(&Config)
	if err != nil {
		Fatal(fmt.Errorf("cannot unmarshal config %v", err))
	}
}

var rootCmd = &cobra.Command{
	Use:   "papy",
	Short: "Papy is a packager and incremental compiler for Skyrim Special Edition mods",
	Long: `A fast and modern packager and incremental compiler for Skyrim Special Edition mods.
Built by the SkyVac team, for SkyVac.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
		os.Exit(0)
	},
}

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// FatalF prints a formatted message to os.stderr and exits with status code 0
func FatalF(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, message, args...)
	os.Exit(0)
}

// Fatal prints an error to stderr and exits with status code 0
func Fatal(err error) {
	FatalF("%v", err)
}

// ifVerbose executes f only if verbose mode is on
func ifVerbose(f func()) {
	if !Verbose {
		return
	}
	f()
}

// VerbosePrintln is like Println, but only if verbose mode is on
func VerbosePrintln(a ...interface{}) { ifVerbose(func() { fmt.Println(a...) }) }

// VerbosePrintf is like PrintF, but only if verbose mode is on
func VerbosePrintf(format string, a ...interface{}) { ifVerbose(func() { fmt.Printf(format, a...) }) }
