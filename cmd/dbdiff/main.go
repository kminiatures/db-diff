package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/koba/db-diff/internal/database"
	"github.com/koba/db-diff/internal/diff"
	"github.com/koba/db-diff/internal/generator"
	"github.com/koba/db-diff/internal/snapshot"
)

var (
	tables    []string
	limit     int
	outputDir string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "dbdiff",
	Short: "Database snapshot and diff tool",
	Long:  `A tool to create database snapshots and compare differences between snapshots.`,
}

var snapshotCmd = &cobra.Command{
	Use:   "snapshot [name]",
	Short: "Create a database snapshot",
	Long:  `Create a snapshot of the current database state.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSnapshot,
}

var diffCmd = &cobra.Command{
	Use:   "diff <snapshot1> <snapshot2>",
	Short: "Compare two snapshots",
	Long:  `Compare two database snapshots and display the differences.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runDiff,
}

var migrateCmd = &cobra.Command{
	Use:   "migrate <snapshot1> <snapshot2>",
	Short: "Generate migration SQL",
	Long:  `Generate DDL and DML statements to migrate from snapshot1 to snapshot2.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runMigrate,
}

func init() {
	// Snapshot command flags
	snapshotCmd.Flags().StringSliceVar(&tables, "tables", nil, "Space-separated list of tables to snapshot (default: all tables)")
	snapshotCmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of rows per table (default: unlimited)")
	snapshotCmd.Flags().StringVar(&outputDir, "output-dir", "./snapshots", "Output directory for snapshots")

	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(migrateCmd)
}

func runSnapshot(cmd *cobra.Command, args []string) error {
	// Load database configuration
	config, err := database.LoadConfigFromEnv()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create database connection
	db, err := database.NewDatabase(config)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Connect to database
	if err := db.Connect(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Generate snapshot filename
	var filename string
	if len(args) > 0 {
		filename = args[0]
		if !strings.HasSuffix(filename, ".db") {
			filename += ".db"
		}
	} else {
		timestamp := time.Now().Format("2006-01-02-15-04-05")
		filename = fmt.Sprintf("%s-%s.db", config.Database, timestamp)
	}

	outputPath := filepath.Join(outputDir, filename)

	// Create snapshot
	fmt.Printf("Creating snapshot: %s\n", outputPath)
	if err := snapshot.CreateSnapshot(db, tables, outputPath, limit); err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	fmt.Printf("Snapshot created successfully: %s\n", outputPath)
	return nil
}

func runDiff(cmd *cobra.Command, args []string) error {
	snapshot1Path := args[0]
	snapshot2Path := args[1]

	// Load snapshots
	fmt.Printf("Loading snapshot: %s\n", snapshot1Path)
	snap1, err := snapshot.LoadSnapshot(snapshot1Path)
	if err != nil {
		return fmt.Errorf("failed to load snapshot1: %w", err)
	}

	fmt.Printf("Loading snapshot: %s\n", snapshot2Path)
	snap2, err := snapshot.LoadSnapshot(snapshot2Path)
	if err != nil {
		return fmt.Errorf("failed to load snapshot2: %w", err)
	}

	// Compare snapshots
	fmt.Printf("\n=== Comparing snapshots ===\n\n")
	result := diff.Compare(snap1, snap2)

	// Display differences
	diff.Display(result)

	return nil
}

func runMigrate(cmd *cobra.Command, args []string) error {
	snapshot1Path := args[0]
	snapshot2Path := args[1]

	// Load snapshots
	snap1, err := snapshot.LoadSnapshot(snapshot1Path)
	if err != nil {
		return fmt.Errorf("failed to load snapshot1: %w", err)
	}

	snap2, err := snapshot.LoadSnapshot(snapshot2Path)
	if err != nil {
		return fmt.Errorf("failed to load snapshot2: %w", err)
	}

	// Compare snapshots
	result := diff.Compare(snap1, snap2)

	// Detect database type from metadata or use default
	dbType := "mysql" // Default, could be enhanced to detect from snapshot metadata

	// Generate migration SQL
	fmt.Printf("-- Migration SQL from %s to %s\n", filepath.Base(snapshot1Path), filepath.Base(snapshot2Path))
	fmt.Printf("-- Generated at: %s\n\n", time.Now().Format(time.RFC3339))

	sql := generator.GenerateSQL(result, dbType)
	fmt.Println(sql)

	return nil
}
