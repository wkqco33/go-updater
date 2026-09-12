package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/wkqco33/go-updater/internal/cli"
	"github.com/wkqco33/go-updater/internal/guenv"
	"github.com/wkqco33/go-updater/internal/versions"
)

var listJSON bool

type listEntry struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type listOutput struct {
	Root     string      `json:"root"`
	Current  string      `json:"current,omitempty"`
	Versions []listEntry `json:"versions"`
}

var listCmd = &cli.Command{
	Use:   "list",
	Short: "설치된 Go 버전 목록을 출력합니다.",
	Long: `로컬에 설치된 모든 Go 버전과 현재 활성 버전을 출력합니다.

예시:
  gu list
  gu list --json

문서: ` + docsURL + `
이슈: ` + issuesURL,
	Args: cli.NoArgs,
	RunE: func(cmd *cli.Command, args []string) error {
		root, err := guenv.Resolve(globals.Home)
		if err != nil {
			return err
		}
		items, err := versions.NewStore(root).List()
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()

		if listJSON {
			return json.NewEncoder(out).Encode(buildListOutput(root, items))
		}
		if len(items) == 0 {
			fmt.Fprintln(out, "설치된 Go 버전이 없습니다. 'gu install' 명령어를 사용하여 설치하세요.")
			return nil
		}
		marker := "*"
		if colorEnabled(out) {
			marker = "\x1b[32m*\x1b[0m"
		}
		fmt.Fprintln(out, "설치된 Go 버전 목록:")
		for _, item := range items {
			if item.Active {
				fmt.Fprintf(out, "  %s %s (활성화됨)\n", marker, item.Name)
				continue
			}
			fmt.Fprintf(out, "    %s\n", item.Name)
		}
		return nil
	},
}

func buildListOutput(root string, items []versions.Version) listOutput {
	output := listOutput{Root: root, Versions: make([]listEntry, 0, len(items))}
	for _, item := range items {
		output.Versions = append(output.Versions, listEntry{Name: item.Name, Active: item.Active})
		if item.Active {
			output.Current = item.Name
		}
	}
	return output
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", "", false, "결과를 JSON으로 출력합니다")
	rootCmd.AddCommand(listCmd)
}
