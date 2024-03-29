package devtools

import (
	"os"
	"path/filepath"

	"github.com/astria/astria-cli-go/cmd"
	"github.com/astria/astria-cli-go/internal/processrunner"
	"github.com/astria/astria-cli-go/internal/ui"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run all the Astria services locally.",
	Long:  `Run all the Astria services locally. This will start the sequencer, cometbft, composer, and conductor.`,
	Run: func(cmd *cobra.Command, args []string) {
		runall()
	},
}

func init() {
	devCmd.AddCommand(runCmd)
}

func runall() {
	ctx := cmd.RootCmd.Context()

	homePath, err := os.UserHomeDir()
	if err != nil {
		log.WithError(err).Error("Error getting home dir")
		panic(err)
	}
	defaultDir := filepath.Join(homePath, ".astria")

	// load the .env file and get the environment variables
	// TODO - move config to own package w/ structs w/ defaults. still use .env for overrides.
	envPath := filepath.Join(defaultDir, "local-dev-astria/.env")
	environment := loadAndGetEnvVariables(envPath)

	// sequencer
	seqOpts := processrunner.NewProcessRunnerOpts{
		Title:   "Sequencer",
		BinPath: filepath.Join(homePath, ".astria/local-dev-astria/astria-sequencer"),
		Env:     environment,
		Args:    nil,
	}
	seqRunner := processrunner.NewProcessRunner(ctx, seqOpts)

	// cometbft
	cometDataPath := filepath.Join(homePath, ".astria/data/.cometbft")
	cometOpts := processrunner.NewProcessRunnerOpts{
		Title:   "Comet BFT",
		BinPath: filepath.Join(homePath, ".astria/local-dev-astria/cometbft"),
		Env:     environment,
		Args:    []string{"node", "--home", cometDataPath},
	}
	cometRunner := processrunner.NewProcessRunner(ctx, cometOpts)

	// composer
	composerOpts := processrunner.NewProcessRunnerOpts{
		Title:   "Composer",
		BinPath: filepath.Join(homePath, ".astria/local-dev-astria/astria-composer"),
		Env:     environment,
		Args:    nil,
	}
	compRunner := processrunner.NewProcessRunner(ctx, composerOpts)

	// conductor
	conductorOpts := processrunner.NewProcessRunnerOpts{
		Title:   "Conductor",
		BinPath: filepath.Join(homePath, ".astria/local-dev-astria/astria-conductor"),
		Env:     environment,
		Args:    nil,
	}
	condRunner := processrunner.NewProcessRunner(ctx, conductorOpts)

	// cleanup function to stop processes if there is an error starting another process
	// FIXME - this isn't good enough. need to use context to stop all processes.
	cleanup := func() {
		seqRunner.Stop()
		cometRunner.Stop()
		compRunner.Stop()
		condRunner.Stop()
	}

	// shouldStart acts as a control channel to start this first process
	shouldStart := make(chan bool)
	close(shouldStart)
	err = seqRunner.Start(shouldStart)
	if err != nil {
		log.WithError(err).Error("Error running sequencer")
		cleanup()
		panic(err)
	}
	err = cometRunner.Start(seqRunner.GetDidStart())
	if err != nil {
		log.WithError(err).Error("Error running cometbft")
		cleanup()
		panic(err)
	}
	err = compRunner.Start(cometRunner.GetDidStart())
	if err != nil {
		log.WithError(err).Error("Error running composer")
		cleanup()
		panic(err)
	}
	err = condRunner.Start(compRunner.GetDidStart())
	if err != nil {
		log.WithError(err).Error("Error running conductor")
		cleanup()
		panic(err)
	}

	runners := []processrunner.ProcessRunner{seqRunner, cometRunner, compRunner, condRunner}

	// create and start ui app
	app := ui.NewApp(runners)
	app.Start()
}
