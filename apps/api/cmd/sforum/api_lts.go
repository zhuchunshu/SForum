package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	apilts "github.com/zhuchunshu/sforum/apps/api/app/Support/APILTS"
)

func newExtensionAPILTSCommand() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "api-lts",
		Short: "Print Host/Frontend API LTS contracts and process-local shim telemetry",
		Long: `Print the seeded Host/Frontend API LTS policy and this process's shim usage.

The running API/worker process records Protocol V1 net/rpc traffic into its own
process-local counters (apilts.Process). This CLI process starts empty unless
you only need the published contract policy (status, RemoveAfter, replacement).

Deletion of sforum.protocol.v1 requires CanRemoveWithZeroShim true for a full
LTS window — see docs/extensions/v3/p13-migration-and-lts.md.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// CLI 默认展示策略种子；若需要本进程 Process 计数则使用同一 Process 单例。
			reg := apilts.Process()
			snap := reg.Snapshot()
			now := time.Now().UTC()
			report := struct {
				apilts.Snapshot
				ProtocolV1Calls            uint64 `json:"protocolV1Calls"`
				ProtocolV1CanRemoveWindow  bool   `json:"protocolV1CanRemoveWindow"`
				ProtocolV1CanRemoveWithZero bool  `json:"protocolV1CanRemoveWithZeroShim"`
				GeneratedAt                string `json:"generatedAt"`
				Note                       string `json:"note"`
			}{
				Snapshot:                   snap,
				ProtocolV1Calls:            reg.ShimCalls(apilts.ProtocolV1ContractID),
				ProtocolV1CanRemoveWindow:  reg.CanRemove(apilts.ProtocolV1ContractID, now),
				ProtocolV1CanRemoveWithZero: reg.CanRemoveWithZeroShim(apilts.ProtocolV1ContractID, now),
				GeneratedAt:                now.Format(time.RFC3339),
				Note: "Live V1 shim counters live in the API/worker process; this CLI process is usually zero.",
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			}
			cmd.Printf("schema: %s\n", snap.SchemaVersion)
			cmd.Printf("minDeprecation: %s\n", snap.MinDeprecation)
			cmd.Printf("protocolV1Calls: %d\n", report.ProtocolV1Calls)
			cmd.Printf("protocolV1CanRemoveWindow: %v\n", report.ProtocolV1CanRemoveWindow)
			cmd.Printf("protocolV1CanRemoveWithZeroShim: %v\n", report.ProtocolV1CanRemoveWithZero)
			cmd.Printf("note: %s\n", report.Note)
			cmd.Println("contracts:")
			for _, c := range snap.Contracts {
				cmd.Printf("  - %s kind=%s status=%s shim=%v replacement=%s\n",
					c.ID, c.Kind, c.Status, c.ShimEnabled, c.Replacement)
			}
			if len(snap.ShimUsage) == 0 {
				cmd.Println("shimUsage: (empty)")
			} else {
				cmd.Println("shimUsage:")
				for _, row := range snap.ShimUsage {
					cmd.Printf("  - %s calls=%d\n", row.ContractID, row.Calls)
				}
			}
			fmt.Fprintln(cmd.OutOrStdout())
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Emit JSON snapshot")
	return cmd
}
