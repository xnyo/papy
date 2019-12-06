package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// Configuration represents the global config file structure
type Configuration struct {
	// CompilerPath is the path to PapyrusCompiler.exe
	CompilerPath string `mapstructure:"compiler_path"`

	// GamePath is the path to the game root folder (where SkyrimSE.exe is)
	GamePath string `mapstructure:"game_path"`
}

func (c Configuration) String() string {
	return fmt.Sprintf("GamePath: %s\nCompilerPath: %s", c.GamePath, c.CompilerPath)
}

// GetSkyrimSEInstallPath returns the Skyrim SE root path from registry.
// if there's no skyrim se path in the registry, it returns an error
func GetSkyrimSEInstallPath() (string, error) {
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

// DiscoverCompiler tries to find the compiler starting from a gamePath.
// it returns the path to PapyrusCompiler.exe if it was found
func DiscoverCompiler(gamePath string) (string, error) {
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

// AutoSetup tries to create a Configuration struct by looking for
// the game path in the registry and then checking for the compiler
func AutoSetup() (*Configuration, error) {
	gamePath, err := GetSkyrimSEInstallPath()
	if err != nil {
		return nil, err
	}
	compilerPath, err := DiscoverCompiler(gamePath)
	if err != nil {
		return nil, err
	}
	return &Configuration{
		CompilerPath: compilerPath,
		GamePath:     gamePath,
	}, nil
}

// ManualSetup asks the user for their skyrim se path
// then it tries to locate PapyrusCompiler.exe in it
// if it's not able to, it asks the user for the path to PapyrusCompiler.exe
func ManualSetup() (*Configuration, error) {
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
	compilerPath, err := DiscoverCompiler(gamePath)
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
