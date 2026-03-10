package cmd

import (
	"github.com/spf13/cobra"
	"shippy/internal/backup"
	"shippy/internal/composer"
	"shippy/internal/ui"
)

var (
	backupVerbose      bool
	backupOutput       string
	backupSkipDatabase bool
	backupSkipShared   bool
)

var backupCmd = &cobra.Command{
	Use:   "backup [host]",
	Short: "Create a backup of database and shared files",
	Long: `Create a ZIP backup containing the database export and shared files
from a remote host.

Database credentials are automatically extracted from the server's
application configuration (.env, TYPO3 settings.php, etc.) or can
be specified manually in .shippy.yaml.

The backup is created locally and includes:
  - database/dump.sql  (database export)
  - shared/            (all shared files and directories)

Example:
  shippy backup                           (interactive host selection)
  shippy backup production
  shippy backup production -o ./backups
  shippy backup production --skip-database`,
	Args:         cobra.MaximumNArgs(1),
	RunE:         runBackup,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.Flags().BoolVarP(&backupVerbose, "verbose", "v", false, "Show detailed output")
	backupCmd.Flags().StringVarP(&backupOutput, "output", "o", "", "Output directory for the backup ZIP")
	backupCmd.Flags().BoolVar(&backupSkipDatabase, "skip-database", false, "Skip database backup")
	backupCmd.Flags().BoolVar(&backupSkipShared, "skip-shared", false, "Skip shared files backup")
}

func runBackup(cmd *cobra.Command, args []string) error {
	out := ui.New()

	out.Info("Loading configuration from: %s", cfgFile)
	cfg, err := loadConfigFile(out)
	if err != nil {
		return err
	}

	cfg.ApplyDefaults()

	if err := cfg.Validate(); err != nil {
		out.Error("Config validation failed: %v", err)
		return err
	}

	hostName, err := selectHostFromArgs(cfg, args, out)
	if err != nil {
		return err
	}

	// Load composer.json for template processing
	composerPath := cfg.GetComposerPath()
	comp, err := composer.Parse(composerPath)
	if err != nil {
		out.Error("Failed to load composer.json from %s: %v", composerPath, err)
		return err
	}

	if err := cfg.ProcessTemplates(comp); err != nil {
		out.Error("Template processing failed: %v", err)
		return err
	}

	b, err := backup.New(cfg, hostName, backupVerbose)
	if err != nil {
		out.Error("Failed to create backup: %v", err)
		return err
	}

	if _, err := b.Backup(backupSkipDatabase, backupSkipShared, backupOutput); err != nil {
		out.Error("Backup failed: %v", err)
		return err
	}

	return nil
}
