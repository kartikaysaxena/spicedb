package cmd

import (
	"errors"
	"fmt"
	"os"

	log "github.com/authzed/spicedb/internal/logging"
	"github.com/authzed/spicedb/pkg/cmd/server"
	cmdutil "github.com/authzed/spicedb/pkg/cmd/server"
	"github.com/authzed/spicedb/pkg/cmd/testserver"
	"github.com/authzed/spicedb/pkg/releases"
	"github.com/jzelinskie/cobrautil/v2"
	"github.com/jzelinskie/cobrautil/v2/cobrazerolog"
	mcobra "github.com/muesli/mango-cobra"
	"github.com/muesli/roff"
	"github.com/spf13/cobra"
)

func RegisterRootFlags(cmd *cobra.Command) error {
	zl := cobrazerolog.New()
	zl.RegisterFlags(cmd.PersistentFlags())
	if err := zl.RegisterFlagCompletion(cmd); err != nil {
		return fmt.Errorf("failed to register zerolog flag completion: %w", err)
	}

	releases.RegisterFlags(cmd.PersistentFlags())

	return nil
}

var ErrParsing = errors.New("parsing error")

// buildRootCommand creates and configures the complete SpiceDB CLI command structure
func BuildRootCommand() (*cobra.Command, error) {
	// Create a root command
	rootCmd := NewRootCommand("spicedb")
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.Println(err)
		cmd.Println(cmd.UsageString())
		return ErrParsing
	})
	if err := RegisterRootFlags(rootCmd); err != nil {
		return nil, fmt.Errorf("failed to register root flags: %w", err)
	}

	// Add a version command
	versionCmd := NewVersionCommand(rootCmd.Use)
	RegisterVersionFlags(versionCmd)
	rootCmd.AddCommand(versionCmd)

	// Add datastore commands
	datastoreCmd, err := NewDatastoreCommand(rootCmd.Use)
	if err != nil {
		return nil, fmt.Errorf("failed to register datastore command: %w", err)
	}

	RegisterDatastoreRootFlags(datastoreCmd)
	rootCmd.AddCommand(datastoreCmd)

	// Add deprecated head command
	headCmd := NewHeadCommand(rootCmd.Use)
	RegisterHeadFlags(headCmd)
	headCmd.Hidden = true
	headCmd.RunE = DeprecatedRunE(headCmd.RunE, "spicedb datastore head")
	rootCmd.AddCommand(headCmd)

	// Add deprecated migrate command
	migrateCmd := NewMigrateCommand(rootCmd.Use)
	migrateCmd.Hidden = true
	migrateCmd.RunE = DeprecatedRunE(migrateCmd.RunE, "spicedb datastore migrate")
	RegisterMigrateFlags(migrateCmd)
	rootCmd.AddCommand(migrateCmd)

	// Add server commands
	serverConfig := cmdutil.NewConfigWithOptionsAndDefaults()
	serveCmd := NewServeCommand(rootCmd.Use, serverConfig)
	if err := RegisterServeFlags(serveCmd, serverConfig); err != nil {
		return nil, fmt.Errorf("failed to register server flags: %w", err)
	}
	rootCmd.AddCommand(serveCmd)

	lspConfig := new(LSPConfig)
	lspCmd := NewLSPCommand(rootCmd.Use, lspConfig)
	if err := RegisterLSPFlags(lspCmd, lspConfig); err != nil {
		return nil, fmt.Errorf("failed to register lsp flags: %w", err)
	}
	rootCmd.AddCommand(lspCmd)

	var testServerConfig testserver.Config
	testingCmd := NewTestingCommand(rootCmd.Use, &testServerConfig)
	RegisterTestingFlags(testingCmd, &testServerConfig)
	rootCmd.AddCommand(testingCmd)

	rootCmd.AddCommand(&cobra.Command{
		Use:                   "man",
		Short:                 "Generate the SpiceDB manpage",
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		Hidden:                true,
		Args:                  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			manPage, err := mcobra.NewManPage(1, cmd.Root())
			if err != nil {
				return err
			}

			_, err = fmt.Fprint(os.Stdout, manPage.Build(roff.NewDocument()))
			return err
		},
	})

	return rootCmd, nil
}

// DeprecatedRunE wraps the RunFunc with a warning log statement.
func DeprecatedRunE(fn cobrautil.CobraRunFunc, newCmd string) cobrautil.CobraRunFunc {
	return func(cmd *cobra.Command, args []string) error {
		log.Warn().Str("newCommand", newCmd).Msg("use of deprecated command")
		return fn(cmd, args)
	}
}

func NewRootCommand(programName string) *cobra.Command {
	return &cobra.Command{
		Use:           programName,
		Short:         "A modern permissions database",
		Long:          "A database that stores, computes, and validates application permissions",
		Example:       server.ServeExample(programName),
		SilenceErrors: true,
		SilenceUsage:  true,
	}
}
