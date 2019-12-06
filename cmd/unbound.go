package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xnyo/papy/papyrus"
)

func init() {
	rootCmd.AddCommand(unboundCmd)
}

// dirents returns all entries (files and folders) in the current folder
func dirents(dir string) ([]os.FileInfo, error) {
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot get directory entities for %s: %v", dir, err)
	}
	return entries, nil
}

func buildDirSet(folder string, ext string) (*map[string]struct{}, error) {
	entries, err := dirents(folder)
	if err != nil {
		return nil, err
	}
	pex := make(map[string]struct{})
	for _, entry := range entries {
		entry := strings.ToLower(entry.Name())
		entryExt := filepath.Ext(entry)
		if entryExt != ext {
			continue
		}
		pex[entry[:len(entry)-len(ext)]] = struct{}{}
	}
	return &pex, nil
}

func setEmptier(set *map[string]struct{}, remove <-chan string) {
	for k := range remove {
		delete(*set, k)
	}
}

func walkWorker(folder string, ext string, remove chan<- string) error {
	entries, err := dirents(folder)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		entry := strings.ToLower(entry.Name())
		entryExt := filepath.Ext(entry)
		if entryExt != ext {
			continue
		}
		remove <- entry[:len(entry)-len(ext)]
	}
	return nil
}

var unboundCmd = &cobra.Command{
	Use:   "unbound",
	Short: "Prints all pex files with no corresponding psc",
	Run: func(cmd *cobra.Command, args []string) {
		// Read yaml
		projectFile := "papy.yaml"
		if len(args) >= 1 {
			projectFile = args[0]
		}
		p, err := papyrus.UnmarshalFile(projectFile, &Config)
		if err != nil {
			Fatal(err)
		}
		pexSet, err := buildDirSet(p.OutputFolder, ".pex")
		if err != nil {
			Fatal(err)
		}
		workers := len(p.Folders)
		done := make(chan struct{}, workers)
		remove := make(chan string, workers)
		go setEmptier(pexSet, remove)
		for _, f := range p.Folders {
			f := f
			go func() {
				err := walkWorker(f, ".psc", remove)
				if err != nil {
					Fatal(err)
				}
				done <- struct{}{}
			}()
		}
		for i := 0; i < workers; i++ {
			<-done
		}
		close(remove)
		for k := range *pexSet {
			fmt.Printf("%s.psc\n", k)
		}
	},
}
