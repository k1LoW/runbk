/*
Copyright © 2022 Ken'ichiro Oyama <k1lowxb@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/k1LoW/donegroup"
	"github.com/k1LoW/duration"
	"github.com/k1LoW/runn"
	"github.com/ryo-yamaoka/otchkiss"
	"github.com/ryo-yamaoka/otchkiss/setting"
	"github.com/spf13/cobra"
)

// loadtCmd represents the loadt command.
var loadtCmd = &cobra.Command{
	Use:     "loadt [PATH_PATTERN]",
	Short:   "run load test using runbooks",
	Long:    `run load test using runbooks.`,
	Aliases: []string{"loadtest"},
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx, cancel := donegroup.WithCancel(context.Background())
		defer func() {
			cancel()
			err = errors.Join(err, donegroup.Wait(ctx))
		}()
		pathp := strings.Join(args, string(filepath.ListSeparator))
		flgs.Format = "none" // Disable runn output
		opts, err := flgs.ToOpts()
		if err != nil {
			return err
		}

		// setup cache dir
		if err := runn.SetCacheDir(flgs.CacheDir); err != nil {
			return err
		}
		defer func() {
			if !flgs.RetainCacheDir {
				err = errors.Join(runn.RemoveCacheDir())
			}
		}()

		o, err := runn.Load(pathp, opts...)
		if err != nil {
			return err
		}
		d, err := duration.Parse(flgs.LoadTDuration)
		if err != nil {
			return err
		}
		w, err := duration.Parse(flgs.LoadTWarmUp)
		if err != nil {
			return err
		}
		s, err := setting.New(flgs.LoadTConcurrent, flgs.LoadTMaxRPS, d, w)
		if err != nil {
			return err
		}
		selected, err := o.SelectedOperators()
		if err != nil {
			return err
		}
		ot, err := otchkiss.FromConfig(o, s, 100_000_000)
		if err != nil {
			return err
		}
		p := tea.NewProgram(newSpinnerModel())
		go func() {
			_, _ = p.Run()
		}()
		if err := ot.Start(ctx); err != nil {
			return err
		}
		p.Quit()
		p.Wait()

		lr, err := runn.NewLoadtResult(len(selected), w, d, flgs.LoadTConcurrent, flgs.LoadTMaxRPS, ot.Result)
		if err != nil {
			return err
		}
		if err := lr.Report(os.Stdout); err != nil {
			return err
		}
		if err := lr.CheckThreshold(flgs.LoadTThreshold); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loadtCmd)
	loadtCmd.Flags().BoolVarP(&flgs.Debug, "debug", "", false, flgs.Usage("Debug"))
	loadtCmd.Flags().BoolVarP(&flgs.FailFast, "fail-fast", "", false, flgs.Usage("FailFast"))
	loadtCmd.Flags().BoolVarP(&flgs.SkipTest, "skip-test", "", false, flgs.Usage("SkipTest"))
	loadtCmd.Flags().BoolVarP(&flgs.SkipIncluded, "skip-included", "", false, flgs.Usage("SkipIncluded"))
	loadtCmd.Flags().StringSliceVarP(&flgs.HostRules, "host-rules", "", []string{}, flgs.Usage("HostRules"))
	loadtCmd.Flags().StringSliceVarP(&flgs.HTTPOpenApi3s, "http-openapi3", "", []string{}, flgs.Usage("HTTPOpenApi3s"))
	loadtCmd.Flags().BoolVarP(&flgs.GRPCNoTLS, "grpc-no-tls", "", false, flgs.Usage("GRPCNoTLS"))
	loadtCmd.Flags().StringSliceVarP(&flgs.GRPCProtos, "grpc-proto", "", []string{}, flgs.Usage("GRPCProtos"))
	loadtCmd.Flags().StringSliceVarP(&flgs.GRPCImportPaths, "grpc-import-path", "", []string{}, flgs.Usage("GRPCImportPaths"))
	loadtCmd.Flags().StringSliceVarP(&flgs.GRPCBufDirs, "grpc-buf-dir", "", []string{}, flgs.Usage("GRPCBufDirs"))
	loadtCmd.Flags().StringSliceVarP(&flgs.GRPCBufLocks, "grpc-buf-lock", "", []string{}, flgs.Usage("GRPCBufLocks"))
	loadtCmd.Flags().StringSliceVarP(&flgs.GRPCBufConfigs, "grpc-buf-config", "", []string{}, flgs.Usage("GRPCBufConfigs"))
	loadtCmd.Flags().StringSliceVarP(&flgs.GRPCBufModules, "grpc-buf-module", "", []string{}, flgs.Usage("GRPCBufModules"))
	loadtCmd.Flags().StringVarP(&flgs.CaptureDir, "capture", "", "", flgs.Usage("CaptureDir"))
	loadtCmd.Flags().StringSliceVarP(&flgs.Vars, "var", "", []string{}, flgs.Usage("Vars"))
	loadtCmd.Flags().StringSliceVarP(&flgs.Runners, "runner", "", []string{}, flgs.Usage("Runners"))
	loadtCmd.Flags().StringSliceVarP(&flgs.Overlays, "overlay", "", []string{}, flgs.Usage("Overlays"))
	loadtCmd.Flags().StringSliceVarP(&flgs.Underlays, "underlay", "", []string{}, flgs.Usage("Underlays"))
	loadtCmd.Flags().StringVarP(&flgs.RunMatch, "run", "", "", flgs.Usage("RunMatch"))
	loadtCmd.Flags().StringSliceVarP(&flgs.RunIDs, "id", "", []string{}, flgs.Usage("RunIDs"))
	loadtCmd.Flags().StringSliceVarP(&flgs.RunLabels, "label", "", []string{}, flgs.Usage("RunLabels"))
	loadtCmd.Flags().IntVarP(&flgs.Sample, "sample", "", 0, flgs.Usage("Sample"))
	loadtCmd.Flags().StringVarP(&flgs.Shuffle, "shuffle", "", "off", flgs.Usage("Shuffle"))
	loadtCmd.Flags().StringVarP(&flgs.Concurrent, "concurrent", "", "off", flgs.Usage("Concurrent"))
	loadtCmd.Flags().IntVarP(&flgs.Random, "random", "", 0, flgs.Usage("Random"))
	loadtCmd.Flags().IntVarP(&flgs.ShardIndex, "shard-index", "", 0, flgs.Usage("ShardIndex"))
	loadtCmd.Flags().IntVarP(&flgs.ShardN, "shard-n", "", 0, flgs.Usage("ShardN"))
	loadtCmd.Flags().StringVarP(&flgs.CacheDir, "cache-dir", "", "", flgs.Usage("CacheDir"))
	loadtCmd.Flags().BoolVarP(&flgs.RetainCacheDir, "retain-cache-dir", "", false, flgs.Usage("RetainCacheDir"))
	loadtCmd.Flags().StringVarP(&flgs.WaitTimeout, "wait-timeout", "", "10sec", flgs.Usage("WaitTimeout"))
	loadtCmd.Flags().StringVarP(&flgs.EnvFile, "env-file", "", "", flgs.Usage("EnvFile"))
	if err := loadtCmd.MarkFlagFilename("env-file"); err != nil {
		panic(err)
	}

	loadtCmd.Flags().IntVarP(&flgs.LoadTConcurrent, "load-concurrent", "", 1, flgs.Usage("LoadTConcurrent"))
	loadtCmd.Flags().StringVarP(&flgs.LoadTDuration, "duration", "", "10sec", flgs.Usage("LoadTDuration"))
	loadtCmd.Flags().StringVarP(&flgs.LoadTWarmUp, "warm-up", "", "5sec", flgs.Usage("LoadTWarmUp"))
	loadtCmd.Flags().StringVarP(&flgs.LoadTThreshold, "threshold", "", "", flgs.Usage("LoadTThreshold"))
	loadtCmd.Flags().IntVarP(&flgs.LoadTMaxRPS, "max-rps", "", 1, flgs.Usage("LoadTMaxRPS"))
}
