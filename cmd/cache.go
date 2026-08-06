package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Snider/Borg/pkg/cache"
	"github.com/spf13/cobra"
)

// cacheCmd represents the cache command
var cacheCmd = NewCacheCmd()
var cacheStatsCmd = NewCacheStatsCmd()
var cacheClearCmd = NewCacheClearCmd()

func init() {
	RootCmd.AddCommand(GetCacheCmd())
	GetCacheCmd().AddCommand(GetCacheStatsCmd())
	GetCacheCmd().AddCommand(GetCacheClearCmd())
}

func GetCacheCmd() *cobra.Command {
	return cacheCmd
}

func GetCacheStatsCmd() *cobra.Command {
	return cacheStatsCmd
}

func GetCacheClearCmd() *cobra.Command {
	return cacheClearCmd
}

func NewCacheCmd() *cobra.Command {
	cacheCmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the cache",
		Long:  `Manage the cache.`,
	}
	return cacheCmd
}

func NewCacheStatsCmd() *cobra.Command {
	cacheStatsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show cache stats",
		Long:  `Show cache stats.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cacheDir, err := GetCacheDir(cmd)
			if err != nil {
				return err
			}
			cacheInstance, err := cache.New(cacheDir)
			if err != nil {
				return fmt.Errorf("failed to create cache: %w", err)
			}

			size, err := cacheInstance.Size()
			if err != nil {
				return fmt.Errorf("failed to get cache size: %w", err)
			}

			fmt.Printf("Cache directory: %s\n", cacheInstance.Dir())
			fmt.Printf("Number of entries: %d\n", cacheInstance.NumEntries())
			fmt.Printf("Total size: %d bytes\n", size)

			return nil
		},
	}
	cacheStatsCmd.Flags().String("cache-dir", "", "Custom cache location")
	return cacheStatsCmd
}

func NewCacheClearCmd() *cobra.Command {
	cacheClearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear the cache",
		Long:  `Clear the cache.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cacheDir, err := GetCacheDir(cmd)
			if err != nil {
				return err
			}
			cacheInstance, err := cache.New(cacheDir)
			if err != nil {
				return fmt.Errorf("failed to create cache: %w", err)
			}

			err = cacheInstance.Clear()
			if err != nil {
				return fmt.Errorf("failed to clear cache: %w", err)
			}

			fmt.Println("Cache cleared.")

			return nil
		},
	}
	cacheClearCmd.Flags().String("cache-dir", "", "Custom cache location")
	return cacheClearCmd
}
