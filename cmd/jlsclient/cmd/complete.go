// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"io/ioutil"
	"os"

	"github.com/bryanl/jsonnet-language-server/pkg/analysis/lexical"
	"github.com/bryanl/jsonnet-language-server/pkg/config"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-jsonnet/ast"

	"github.com/spf13/cobra"
)

// completeCmd represents the complete command
var completeCmd = &cobra.Command{
	Use:   "complete",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		/* #nosec */
		f, err := os.Open(completeFilename)
		if err != nil {
			return err
		}

		cfg := config.New()
		update := map[string]interface{}{
			config.JsonnetLibPaths: completeJLibPaths,
		}

		if err = cfg.UpdateClientConfiguration(update); err != nil {
			return err
		}

		data, err := ioutil.ReadFile(completeFilename)
		if err != nil {
			return err
		}

		uriStr := "file://" + completeFilename

		td := config.NewTextDocument(uriStr, string(data))
		if err = cfg.StoreTextDocumentItem(td); err != nil {
			return err
		}

		response, err := lexical.CompletionAtLocation(
			uriStr,
			f,
			ast.Location{Line: completeLine, Column: completeCol},
			cfg,
		)
		if err != nil {
			return err
		}

		spew.Dump(response)

		return nil
	},
}

var (
	completeFilename  string
	completeLine      int
	completeCol       int
	completeJLibPaths []string
)

func init() {
	rootCmd.AddCommand(completeCmd)

	completeCmd.Flags().StringVarP(&completeFilename, "filename", "f", "", "filename")
	completeCmd.Flags().IntVarP(&completeLine, "line", "l", 0, "line")
	completeCmd.Flags().IntVarP(&completeCol, "column", "c", 0, "column")
	completeCmd.Flags().StringSliceVarP(&completeJLibPaths, "jpath", "j", []string{}, "jsonnet lib paths")
}
