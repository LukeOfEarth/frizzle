package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initSimple bool

func init() {
	initCmd.Flags().BoolVar(&initSimple, "simple", false, "Generate a minimal single-bus config")
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a .frizzle/frizzle.json config file",
	Long: `Creates a .frizzle/frizzle.json config file in the current directory.

By default, it generates a full example config with multiple buses, rules,
and target types to show what's possible.

Use --simple to generate a minimal config with a single catch-all bus.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := filepath.Join(".frizzle", "frizzle.json")

		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf(".frizzle/frizzle.json already exists — delete it first if you want to regenerate")
		}

		cfg := fullConfig
		if initSimple {
			cfg = simpleConfig
		}

		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		if err := os.MkdirAll(".frizzle", 0755); err != nil {
			return fmt.Errorf("failed to create .frizzle directory: %w", err)
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write .frizzle/frizzle.json: %w", err)
		}

		fmt.Println("Created .frizzle/frizzle.json")
		return nil
	},
}

var simpleConfig = map[string]interface{}{
	"port": 4000,
	"buses": []map[string]interface{}{
		{
			"name": "default",
			"rules": []map[string]interface{}{
				{
					"name":        "catch-all",
					"description": "Forward all events to the order service and log them",
					"pattern":     map[string]interface{}{},
					"targets": []map[string]interface{}{
						{
							"type": "log",
						},
						{
							"type":   "http",
							"url":    "http://localhost:3001/webhook",
							"method": "POST",
						},
					},
				},
			},
		},
	},
}

var fullConfig = map[string]interface{}{
	"port": 4000,
	"buses": []map[string]interface{}{
		{
			"name": "default",
			"rules": []map[string]interface{}{
				{
					"name":        "order-events",
					"description": "Forward order events to the order service",
					"pattern": map[string]interface{}{
						"source":      []string{"myapp.orders", "myapp.checkout"},
						"detail-type": []string{"OrderPlaced", "OrderCancelled"},
					},
					"targets": []map[string]interface{}{
						{
							"type": "log",
						},
						{
							"type":    "http",
							"url":     "http://localhost:3001/webhook",
							"method":  "POST",
							"timeout": "5s",
						"headers": map[string]string{
							"X-Source": "frizzle",
						},
						},
					},
				},
				{
					"name":        "high-value-orders",
					"description": "Alert on orders above $500",
					"pattern": map[string]interface{}{
						"detail-type": []string{"OrderPlaced"},
						"detail": map[string]interface{}{
							"amount": []map[string]interface{}{
								{"numeric": []interface{}{">=", 500}},
							},
						},
					},
					"targets": []map[string]interface{}{
						{
							"type": "log",
						},
						{
							"type":   "http",
							"url":    "http://localhost:3002/alerts",
							"method": "POST",
						},
					},
				},
				{
					"name":        "catch-all",
					"description": "Log any events that don't match other rules",
					"pattern":     map[string]interface{}{},
					"targets": []map[string]interface{}{
						{
							"type": "log",
						},
					},
				},
			},
		},
		{
			"name": "analytics",
			"rules": []map[string]interface{}{
				{
					"name":        "user-events",
					"description": "Forward user-related events to analytics pipeline",
					"pattern": map[string]interface{}{
						"source": []map[string]interface{}{
							{"prefix": "myapp.users"},
						},
					},
					"targets": []map[string]interface{}{
						{
							"type": "log",
						},
						{
							"type":   "http",
							"url":    "http://localhost:3003/analytics",
							"method": "POST",
						},
					},
				},
			},
		},
	},
}
