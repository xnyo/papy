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

	"github.com/spf13/cobra"
	"golang.org/x/sys/windows/registry"
)

func init() {
	rootCmd.AddCommand(setupCmd)
}

// getSkyrimSEInstallPath returns the Skyrim SE root path from registry.
// if there's no skyrim se path in the registry, it returns an error
func getSkyrimSEInstallPath() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Bethesda Softworks\Skyrim Special Edition`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()
	v, _, err := k.GetStringValue("Installed Path")
	if err != nil {
		return "", err
	}
	return v, nil
}

// discoverCompiler tries to find the compiler starting from a gamePath.
// it returns the path to PapyrusCompiler.exe if it was found
func discoverCompiler(gamePath string) (string, error) {
	compilerPath := filepath.Join(gamePath, "Papyrus Compiler", "PapyrusCompiler.exe")
	s, err := os.Stat(compilerPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf(`cannot find %s\Papyrus Compiler\PapyrusCompiler.exe`, gamePath)
		}
		return "", err
	}
	if s.IsDir() {
		return "", fmt.Errorf(
			"expected compiler path %s is a directory, it should be a file",
			compilerPath,
		)
	}
	return compilerPath, nil
}

// autoSetup tries to create a Configuration struct by looking for
// the game path in the registry and then checking for the compiler
func autoSetup() (*Configuration, error) {
	gamePath, err := getSkyrimSEInstallPath()
	if err != nil {
		return nil, err
	}
	compilerPath, err := discoverCompiler(gamePath)
	if err != nil {
		return nil, err
	}
	return &Configuration{
		CompilerPath: compilerPath,
		GamePath:     gamePath,
	}, nil
}

// manualSetup asks the user for their skyrim se path
// then it tries to locate PapyrusCompiler.exe in it
// if it's not able to, it asks the user for the path to PapyrusCompiler.exe
func manualSetup() (*Configuration, error) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Path to your Skyrim root folder (where SkyrimSE.exe is): ")
	scanner.Scan()
	gamePath := scanner.Text()
	s, err := os.Stat(gamePath)
	if err != nil {
		return nil, fmt.Errorf("wrong game path: %v", err)
	}
	if !s.IsDir() {
		return nil, fmt.Errorf("game path must be a directory, %v is not a directory", gamePath)
	}
	compilerPath, err := discoverCompiler(gamePath)
	if err != nil {
		fmt.Print("Path to PapyrusCompiler.exe: ")
		scanner.Scan()
		compilerPath = scanner.Text()
		s, err = os.Stat(compilerPath)
		if err != nil {
			return nil, err
		}
		if s.IsDir() {
			return nil, fmt.Errorf("compiler %s must be a file, not a directory", compilerPath)
		}
	}
	return &Configuration{
		CompilerPath: compilerPath,
		GamePath:     gamePath,
	}, nil
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
		var c *Configuration
		fmt.Println("Trying to run auto setup.")
		c, err = autoSetup()
		if err != nil {
			// Manual setup
			fmt.Fprintln(os.Stderr, "Could not run auto setup.")
			c, err = manualSetup()
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
