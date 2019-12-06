package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/xnyo/papy/config"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(setupCmd)
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Creates the global config file",
	Run: func(cmd *cobra.Command, args []string) {
		var configDoesNotExist bool
		usr, err := user.Current()
		if err != nil {
			Fatal(err)
		}
		// Detect existing config file
		configPath := filepath.Join(usr.HomeDir, ".papy.yaml")
		s, err := os.Stat(configPath)
		if err != nil && !os.IsNotExist(err) {
			// Error
			Fatal(err)
		} else if s != nil && s.IsDir() {
			// Directory
			Fatal(errors.New("config does not exist, directory instead"))
		} else if err == nil {
			// Config file already exists
			fmt.Print("Config file already exists. Overwrite? [yN]: ")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			yn := strings.ToLower(strings.Trim(scanner.Text(), " "))
			if yn != "y" {
				os.Exit(0)
			}
		} else {
			// No config file
			configDoesNotExist = true
		}

		// Autosetup
		var c *config.Configuration
		fmt.Println("Trying to run auto setup.")
		c, err = config.AutoSetup()
		if err != nil {
			// Manual setup
			fmt.Fprintln(os.Stderr, "Could not run auto setup.")
			c, err = config.ManualSetup()
			if err != nil {
				Fatal(err)
			}
		}

		// Setup ok
		fmt.Println(c)
		viper.Set("compiler_path", c.CompilerPath)
		viper.Set("game_path", c.GamePath)

		// Viper does not create the config file for some reason
		if configDoesNotExist {
			func() {
				f, err := os.Create(configPath)
				defer f.Close()
				if err != nil {
					Fatal(fmt.Errorf("cannot create empty config file: %v", err))
				}
			}()
		}
		err = viper.WriteConfig()
		if err != nil {
			FatalF("error while writing config file: %v", err)
		}
		fmt.Println("ok")
	},
}
