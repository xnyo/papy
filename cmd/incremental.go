package cmd

import (
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/spf13/cobra"
	"github.com/xnyo/papy/papyrus"
)

// Number of workers, -w flag
var workers int

func init() {
	incrementalCmd.Flags().IntVarP(&workers, "workers", "w", 0, "number of workers. 0 for cpu cores.")
	rootCmd.AddCommand(incrementalCmd)
}

var incrementalCmd = &cobra.Command{
	Use:   "incremental [project_file]",
	Short: "Compiles all new scripts or that have been edited",
	Args:  cobra.MaximumNArgs(1),
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

		// Check folders
		if err = p.CheckFolders(); err != nil {
			Fatal(err)
		}

		// Check compiler
		if s, err := os.Stat(Config.CompilerPath); err != nil || s.IsDir() {
			FatalF("compiler check error: %v", err)
		}
		VerbosePrintf("Using compiler %s\n", Config.CompilerPath)

		// Figure out which scripts to compile
		r, err := p.GetScriptsToCompile()
		if err != nil {
			Fatal(err)
		}
		numberOfScripts := len(*r)
		VerbosePrintf("Going to compile %d scripts.\n", numberOfScripts)

		// Determine workers
		if workers <= 0 {
			workers = runtime.NumCPU()
		}
		VerbosePrintf("Using %d workers\n", workers)

		// Spawn workers
		files := make(chan string, workers)
		results := make(chan *papyrus.CompilerResult, workers)
		var wg sync.WaitGroup
		wg.Add(workers)
		for i := 0; i < workers; i++ {
			go p.CompileWorker(Config, &wg, files, results)
		}

		// Error reporting goroutine
		go func() {
			for result := range results {
				if result.Err != nil {
					fmt.Fprintf(
						os.Stderr,
						"Error while compiling %s:\n%v\n%s",
						result.SourceScript,
						result.Err,
						result.Output,
					)
				}
			}
		}()

		// Send all files to workers
		for _, file := range *r {
			files <- file
		}
		close(files)

		// Wait for all workers to finish
		wg.Wait()

		// This will stop the error reporting goroutine
		close(results)

		VerbosePrintln("Done!")
	},
}
