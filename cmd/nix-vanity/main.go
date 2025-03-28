package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nix-community/go-nix/pkg/derivation"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/exp/slog"
)

// lookupDrvReplacementFromFileSystem remains largely the same, but ensure memoization is handled correctly.
// It's crucial that the memoize map is shared across all recursive calls initiated
// for a single top-level derivation's inputs.
func lookupDrvReplacementFromFileSystem(memoize map[string]string) func(string) (string, error) {
	// This recursive function needs to capture the memoize map
	var lookupFunc func(string) (string, error)
	lookupFunc = func(drvPath string) (string, error) {
		if memoized, found := memoize[drvPath]; found {
			return memoized, nil
		}

		f, err := os.Open(drvPath)
		if err != nil {
			// Wrap error for context
			return "", fmt.Errorf("opening drv %q: %w", drvPath, err)
		}
		defer f.Close()

		drv, err := derivation.ReadDerivation(f)
		if err != nil {
			return "", fmt.Errorf("reading drv %q: %w", drvPath, err)
		}

		// Pass the *same* lookupFunc (which captures the memoize map) recursively
		replacement, err := drv.CalculateDrvReplacementRecursive(lookupFunc)
		if err != nil {
			return "", fmt.Errorf("calculating replacement for drv %q: %w", drvPath, err)
		}

		// memoize the result
		memoize[drvPath] = replacement
		return replacement, nil
	}
	return lookupFunc
}

// result holds the successful seed and the resulting derivation
type result struct {
	seed string
	drv  *derivation.Derivation
}

func main() {
	// --- Configuration ---
	var (
		derivationPath string
		prefix         string
		numWorkers     int
		outputName     string
		seed           uint64
	)

	flag.Uint64Var(&seed, "seed", 0, "Initial seed for the random number generator (default: 0)")
	flag.StringVar(&prefix, "prefix", "", "Desired prefix for the 'out' output path (e.g., /nix/store/abc)")
	flag.IntVar(&numWorkers, "workers", runtime.NumCPU(), "Number of concurrent workers")
	flag.StringVar(&outputName, "output", "out", "Name of the output path to check for the prefix")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <path-to-derivation>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		slog.Error("Missing required argument: <path-to-derivation>")
		flag.Usage()
		os.Exit(1)
	}
	derivationPath = flag.Arg(0)

	if prefix == "" {
		slog.Error("Missing required flag: -prefix")
		flag.Usage()
		os.Exit(1)
	}
	if !strings.HasPrefix(prefix, "/nix/store/") {
		slog.Error("Prefix does not start with /nix/store/", "prefix", prefix)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})) // Use Info level for less noise
	slog.SetDefault(logger)

	// --- Load Base Derivation ---
	slog.Info("Loading base derivation", "path", derivationPath)
	baseDrvFile, err := os.Open(derivationPath)
	if err != nil {
		slog.Error("Error opening base derivation file", "path", derivationPath, "error", err)
		os.Exit(1)
	}
	defer baseDrvFile.Close()

	baseDrv, err := derivation.ReadDerivation(baseDrvFile)
	if err != nil {
		slog.Error("Error reading base derivation", "path", derivationPath, "error", err)
		os.Exit(1)
	}

	// Ensure Env is initialized
	if baseDrv.Env == nil {
		baseDrv.Env = make(map[string]string)
	}

	// --- Calculate Input Derivation Replacements (Done Once) ---
	slog.Info("Calculating input derivation replacements...")
	// Use a single memoization map for all lookups related to the base derivation's inputs
	inputMemoize := make(map[string]string, len(baseDrv.InputDerivations)*2)
	lookupFunc := lookupDrvReplacementFromFileSystem(inputMemoize)
	drvReplacements := make(map[string]string, len(baseDrv.InputDerivations))

	for inputDrvPath := range baseDrv.InputDerivations {
		// Note: We don't need to read the input drv file here again.
		// CalculateDrvReplacementRecursive handles the recursion via the lookupFunc.
		replacement, err := lookupFunc(inputDrvPath)
		if err != nil {
			slog.Error("Error calculating input replacement", "error", err)
			os.Exit(1)
		}
		drvReplacements[inputDrvPath] = replacement
		slog.Debug("Calculated replacement", "input", inputDrvPath, "replacement", replacement)
	}
	slog.Info("Finished calculating input derivation replacements.")

	// --- Setup Concurrent Search ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancellation signal is sent on exit

	var wg sync.WaitGroup
	resultChan := make(chan result, 1)          // Buffered channel to hold the first result
	seedChan := make(chan uint64, numWorkers*2) // Channel to distribute seeds
	var attempts atomic.Uint64                  // Atomic counter for attempts
	slog.Info("Starting workers", "count", numWorkers)

	// Progress Bar
	// Use -1 for max to indicate unknown duration, updating manually
	bar := progressbar.NewOptions64(-1,
		progressbar.OptionSetDescription("Searching for prefix..."),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWidth(15),
		progressbar.OptionThrottle(100*time.Millisecond), // Update interval
		progressbar.OptionShowCount(),
		progressbar.OptionShowTotalBytes(false),
		progressbar.OptionSetItsString("drv"),
		progressbar.OptionShowIts(), // Show iterations per second
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)

	// --- Start Workers ---
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			slog.Debug("Worker started", "id", workerID)

			// Each worker needs its own *copy* of the environment map
			// derived from the base derivation to avoid race conditions.
			// We create a slightly modified derivation copy inside the loop.

			for {
				select {
				case <-ctx.Done(): // Check for cancellation signal
					slog.Debug("Worker cancelling", "id", workerID)
					return
				case seed, ok := <-seedChan:
					if !ok {
						slog.Debug("Seed channel closed, worker stopping", "id", workerID)
						return
					}

					// Create a shallow copy of the base derivation for this attempt
					currentDrv := *baseDrv
					// Create a *new* environment map for this attempt
					currentEnv := make(map[string]string, len(baseDrv.Env)+1)
					for k, v := range baseDrv.Env {
						currentEnv[k] = v
					}
					seedStr := strconv.FormatUint(seed, 10)
					currentEnv["VANITY_SEED"] = seedStr
					currentDrv.Env = currentEnv // Assign the unique env map

					// Calculate output paths using the modified derivation copy
					outputs, err := currentDrv.CalculateOutputPaths(drvReplacements)
					count := attempts.Add(1) // Increment attempt counter atomically
					bar.Set64(int64(count))  // Update progress bar

					if err != nil {
						// Log error but continue searching, might be transient? Or maybe stop?
						// For now, log and continue. If this is common, might need rethinking.
						slog.Warn("Error calculating output paths for seed", "seed", seedStr, "error", err)
						continue // Try next seed
					}

					// Check if the desired output path has the prefix
					outputPath, found := outputs[outputName]
					if !found {
						// This should not happen if the derivation is valid, maybe exit?
						slog.Error("Output name not found in calculated outputs", "output_name", outputName, "seed", seedStr)
						// Optionally: cancel() here if this is critical
						continue
					}

					if strings.HasPrefix(outputPath, prefix) {
						slog.Info("Prefix found!", "seed", seedStr, "output_name", outputName, "path", outputPath)

						// Prepare the final derivation object with the correct outputs
						finalDrv := currentDrv // Start with the drv copy that worked
						// Update Outputs map and Env map with ALL calculated outputs
						for name, path := range outputs {
							if _, ok := finalDrv.Outputs[name]; ok {
								finalDrv.Outputs[name].Path = path
							} else {
								// This case might indicate an issue, but handle defensively
								slog.Warn("Output name present in calculation but not in drv.Outputs map", "name", name)
								// Decide if you want to add it or ignore
							}
							finalDrv.Env[name] = path // Ensure env var is also set
						}

						// Try sending the result. If channel is full/closed, another worker won.
						select {
						case resultChan <- result{seed: seedStr, drv: &finalDrv}:
							slog.Debug("Worker sent result", "id", workerID)
							cancel() // Signal all other workers and seed generator to stop
						case <-ctx.Done():
							// Context was cancelled while trying to send, another worker won.
							slog.Debug("Context cancelled before worker could send result", "id", workerID)
						}
						return // This worker is done
					}
					// else: Prefix not matched, continue loop
				}
			}
		}(i)
	}

	// --- Start Seed Generator ---
	go func() {
		for {
			select {
			case <-ctx.Done(): // Stop generating if context is cancelled
				slog.Debug("Seed generator stopping")
				close(seedChan) // Close channel to signal workers no more seeds are coming
				return
			case seedChan <- seed:
				seed++
				// Handle potential overflow if you run this for a *very* long time
				if seed == 0 { // Check if wrapped around
					slog.Warn("Seed counter overflowed!")
					// Optionally stop or reset, depending on desired behavior
				}
			}
		}
	}()

	// --- Wait for Result or Completion ---
	finalResult := <-resultChan
	// Success!
	bar.Finish() // Mark progress bar as complete
	slog.Info("Successfully found seed", "seed", finalResult.seed)

	// Write the successful derivation to stdout
	slog.Info("Writing successful derivation to stdout...")
	if err := finalResult.drv.WriteDerivation(os.Stdout); err != nil {
		slog.Error("Error writing final derivation", "error", err)
		os.Exit(1)
	}

	// Wait for all workers to finish cleanly after cancellation
	wg.Wait()
	slog.Info("All workers finished.")
}
