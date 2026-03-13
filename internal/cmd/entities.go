package cmd

import (
	"context"
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/microsoft/durabletask-scheduler/cli/internal/api"
)

func newEntitiesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "entities",
		Aliases: []string{"ent"},
		Short:   "Manage entities",
	}

	cmd.AddCommand(
		newEntityListCmd(),
		newEntityGetCmd(),
		newEntityStateCmd(),
		newEntityDeleteCmd(),
	)

	return cmd
}

// --- list ---

func newEntityListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List entities with optional filters",
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}

			req := &api.QueryEntitiesRequest{
				FetchTotalCount: true,
			}

			// Name filter
			name, _ := c.Flags().GetString("name")
			if name != "" {
				if req.Filter == nil {
					req.Filter = &api.EntityFilter{}
				}
				req.Filter.Name = &api.StringFilter{Value: name}
			}

			// Name starts with
			namePrefix, _ := c.Flags().GetString("name-starts-with")
			if namePrefix != "" {
				if req.Filter == nil {
					req.Filter = &api.EntityFilter{}
				}
				req.Filter.NameStartsWith = &api.StringFilter{Value: namePrefix}
			}

			// Pagination
			pageSize, _ := c.Flags().GetInt("page-size")
			startIndex, _ := c.Flags().GetInt("start-index")
			req.Pagination = &api.Pagination{
				StartIndex: startIndex,
				Count:      pageSize,
			}

			// Sort
			req.Sort = []api.SortOption{
				{Column: api.SortEntityByLastModifiedAt, Direction: api.SortDescending},
			}

			result, err := client.QueryEntities(context.Background(), req)
			if err != nil {
				exitError("query entities failed", err)
			}
			printJSON(result)
		},
	}

	cmd.Flags().String("name", "", "Filter by entity name (substring match)")
	cmd.Flags().String("name-starts-with", "", "Filter by entity name prefix")
	cmd.Flags().Int("page-size", 50, "Number of results per page")
	cmd.Flags().Int("start-index", 0, "Pagination start index")

	return cmd
}

// --- get ---

func newEntityGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <instance-id>",
		Short: "Get entity metadata",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			result, err := client.GetEntity(context.Background(), args[0])
			if err != nil {
				exitError("get entity failed", err)
			}
			printJSON(result)
		},
	}
}

// --- state ---

func newEntityStateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "state <instance-id>",
		Short: "Get the serialized state of an entity",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			raw, err := client.GetEntityState(context.Background(), args[0])
			if err != nil {
				exitError("get entity state failed", err)
			}
			// Try to output as structured JSON; fall back to string wrapper
			var parsed interface{}
			if jsonErr := jsonUnmarshalRaw([]byte(raw), &parsed); jsonErr == nil {
				printJSON(parsed)
			} else {
				printJSON(map[string]string{"state": raw})
			}
		},
	}
}

// --- delete ---

func newEntityDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <instance-id> [instance-id...]",
		Short: "Delete one or more entities",
		Args:  cobra.MinimumNArgs(1),
		Run: func(c *cobra.Command, args []string) {
			client, err := initClient(c)
			if err != nil {
				exitError("init failed", err)
			}
			if len(args) == 1 {
				if err := client.DeleteEntity(context.Background(), args[0]); err != nil {
					exitError("delete entity failed", err)
				}
				printStatus("deleted", args[0])
			} else {
				if err := client.DeleteEntities(context.Background(), args); err != nil {
					exitError("delete entities failed", err)
				}
				printJSON(map[string]interface{}{
					"status": "deleted",
					"ids":    args,
				})
			}
		},
	}
}

// jsonUnmarshalRaw is a thin wrapper for json.Unmarshal used in this package.
func jsonUnmarshalRaw(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
