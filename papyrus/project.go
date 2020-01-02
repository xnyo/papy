package papyrus

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xnyo/papy/config"
	"gopkg.in/yaml.v2"
)

// SourceScript represents a script that needs to be compiled
type SourceScript struct {
	SourcePath        string
	DestinationFolder string
}

// CompilerResult represents the result of a papyrus script compilation
type CompilerResult struct {
	SourceScript *SourceScript
	Command      string
	Output       string
	Err          error
}

// arg is a command line argument that implements Stringer
type arg struct {
	name  string
	value string
}

func (a arg) String() string {
	return fmt.Sprintf("-%s=%s", a.name, a.value)
}

// Project represents a papy yaml project file
type Project struct {
	// OutputFolders is the path to the output folder
	OutputFolders []string `yaml:"output_folders"`

	// Optimize is true if we want to optimize our scripts with the -o flag
	Optimize bool

	// Imports is a slice of strings containing the paths to the folders we want to import (-i flag)
	Imports []string

	// Folders is a slice of strings containing the paths of the folders we want to compile
	Folders []string
}

// UnmarshalFile takes a path to a yaml file and tries
// to unmarshal its content to a Project struct
func UnmarshalFile(inputFileName string, config *config.Configuration) (*Project, error) {
	data, err := ioutil.ReadFile(inputFileName)
	if err != nil {
		return nil, fmt.Errorf("cannot open file %s: %v", inputFileName, err)
	}
	newProject := Project{}
	err = yaml.UnmarshalStrict(data, &newProject)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal project: %v", err)
	}

	// Add source folder to -i imports or it won't compile anything related to
	// any scripts in the same folder
	newProject.addSourceToImports()
	newProject.resolveSpecialPaths(*config)

	// Turn all paths to absolute paths
	err = newProject.absPaths()
	if err != nil {
		return nil, err
	}
	return &newProject, nil
}

// resolveSpecialPaths resolves all special paths
// (right now, only $base_game in Imports)
func (p *Project) resolveSpecialPaths(config config.Configuration) {
	for i, path := range p.Imports {
		if path == "$base_game" {
			p.Imports[i] = filepath.Join(config.GamePath, "Data", "Source", "Scripts")
		}
	}
}

// absPaths turns all relative paths to absolute paths
func (p *Project) absPaths() error {
	for i := 0; i < len(p.Folders); i++ {
		v, err := filepath.Abs(p.Folders[i])
		if err != nil {
			return err
		}
		p.Folders[i] = v
	}
	for i := 0; i < len(p.Imports); i++ {
		v, err := filepath.Abs(p.Imports[i])
		if err != nil {
			return err
		}
		p.Imports[i] = v
	}
	for i := 0; i < len(p.OutputFolders); i++ {
		v, err := filepath.Abs(p.OutputFolders[i])
		if err != nil {
			return err
		}
		p.OutputFolders[i] = v
	}
	return nil
}

// addSourceToImports adds all folders p.Folders to p.Imports, if they're not present
func (p *Project) addSourceToImports() {
	var toAdd []string
	for _, sourceFolder := range p.Folders {
		if !contains(p.Imports, sourceFolder) {
			toAdd = append(toAdd, sourceFolder)
		}
	}
	p.Imports = append(p.Imports, toAdd...)
}

// CheckFolders makes sure that all folders in the project exists
// (imports, output and input)
func (p *Project) CheckFolders() error {
	for _, folder := range p.Folders {
		checkFolder(folder)
	}
	for _, folder := range p.Imports {
		checkFolder(folder)
	}
	for _, folder := range p.OutputFolders {
		checkFolder(folder)
	}
	return nil
}

// GetScriptsToCompile returns all scripts that need to be compiled
func (p *Project) GetScriptsToCompile() (*[]SourceScript, error) {
	var sourceFiles []SourceScript
	for _, inputFolder := range p.Folders {
		r, err := p.walkSourceDir(inputFolder)
		if err != nil {
			return nil, err
		}
		sourceFiles = append(sourceFiles, *r...)
	}
	return &sourceFiles, nil
}

// CompileWorker starts a papyrus compiler to compile
// scripts received from the "c" channel.
// It reports results in the "results" channel.
func (p *Project) CompileWorker(config config.Configuration, wg *sync.WaitGroup, c <-chan *SourceScript, results chan<- *CompilerResult) {
	defer wg.Done()
	for sourceFile := range c {
		fmt.Printf("Compiling %s -> %s\n", filepath.Base(sourceFile.SourcePath), sourceFile.DestinationFolder)
		args := []string{
			sourceFile.SourcePath,
			arg{"o", sourceFile.DestinationFolder}.String(),
			arg{"i", strings.Join(p.Imports, ";")}.String(),
			arg{
				"f",
				"TESV_Papyrus_Flags.flg",
			}.String(),
		}
		if p.Optimize {
			args = append(args, "-o")
		}
		compilerCmd := exec.Command(config.CompilerPath, args...)
		compilerOut, err := compilerCmd.CombinedOutput()
		results <- &CompilerResult{
			SourceScript: sourceFile,
			Command:      strings.Join(args, " "),
			Err:          err,
			Output:       string(compilerOut),
		}
	}
}

func (p *Project) walkSourceDir(dir string) (*[]SourceScript, error) {
	var result []SourceScript
	entries, err := dirents(dir)
	if err != nil {
		return nil, err
	}
	for _, pscInfo := range entries {
		pscFileName := pscInfo.Name()
		var foundPexDir string
		var foundPexInfo os.FileInfo
		if pscInfo.IsDir() {
			// Folder, walk recursively
			// Skyrim has all scripts in one folder, so no.
			/*subdir := filepath.Join(dir, pscInfo.Name())
			subEntries, err := p.walkSourceDir(subdir)
			if err != nil {
				return nil, err
			}
			result = append(result, *subEntries...)*/
			continue
		} else if filepath.Ext(pscFileName) == ".psc" {
			// .psc file, check if we should rebuild this
			pexFileName := pscFileName[:len(pscFileName)-4] + ".pex"
			for _, of := range p.OutputFolders {
				pexPath := filepath.Join(of, pexFileName)
				pexInfo, err := os.Stat(pexPath)
				if err != nil && !os.IsNotExist(err) {
					return nil, fmt.Errorf("cannot stat file %s: %v", pexPath, err)
				} else if pexInfo != nil {
					if pexInfo.IsDir() {
						return nil, fmt.Errorf("%s is dir, expected filr", pexPath)
					} else {
						foundPexDir = pexPath
						foundPexInfo = pexInfo
						break
					}
				}
			}

			if foundPexInfo == nil {
				// .pex does not exist in any folders, it needs to be built!
				// send it to the primary folder
				result = append(result, SourceScript{
					filepath.Join(dir, pscFileName),
					p.OutputFolders[0],
				})
				continue
			}
			if pscInfo.ModTime().After(foundPexInfo.ModTime()) {
				// File modified, it needs to be rebuilt!
				result = append(result, SourceScript{
					filepath.Join(dir, pscFileName),
					filepath.Dir(foundPexDir),
				})
			}
		}
	}
	return &result, nil
}

// dirents returns all entries (files and folders) in the current folder
func dirents(dir string) ([]os.FileInfo, error) {
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot get directory entities for %s: %v", dir, err)
	}
	return entries, nil
}

func folderExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("cannot stat path %s: %v", path, err)
	}
	return info.Mode().IsDir(), nil
}

func checkFolder(folder string) error {
	if folder == "" {
		return fmt.Errorf("cannot have empty folder")
	}
	r, err := folderExists(folder)
	if err != nil {
		return err
	}
	if !r {
		return fmt.Errorf("folder %s does not exist or is a file", folder)
	}
	return nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
