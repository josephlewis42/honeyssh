/*
Copyright © 2021 Joseph Lewis <joseph@josephlewis.net>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	containerregistry "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"
)

// WhiteoutPrefix prefix means file is a whiteout.
const WhiteoutPrefix = ".wh."

// img2fs converts a Docker image to a filesystem
var img2fs = &cobra.Command{
	Use:   "img2fs INPUT_TAR OUTPUT_TAR [TAG]",
	Short: "Convert a docker image to a .tar for use as a root filesystem.",
	Long: `Convert a docker image to a .tar for use as a root filesystem.

Prepare an image by running the following:

	docker pull some-image:latest
	docker save some-image:latest > some-image.tar
	osshit img2fs some-image.tar fs.tar
`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		inputPath := args[0]
		outputPath := args[1]

		// Find the tag associated with the image.
		var tag name.Tag
		if len(args) == 3 {
			var err error
			tag, err = name.NewTag(args[2])
			if err != nil {
				return err
			}
		} else {
			manifest, err := tarball.LoadManifest(func() (io.ReadCloser, error) {
				return os.Open(inputPath)
			})
			if err != nil {
				return err
			}

			if len(manifest) != 1 {
				var tags []string
				for _, m := range manifest {
					tags = append(tags, m.RepoTags...)
				}

				return fmt.Errorf("Multiple tags found in the input, specify one of: %q", tags)
			}
			tag, err = name.NewTag(manifest[0].RepoTags[0])
			if err != nil {
				return err
			}
		}

		image, err := tarball.ImageFromPath(args[0], &tag)
		if err != nil {
			return err
		}

		layers, err := image.Layers()
		if err != nil {
			return err
		}

		out, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer out.Close()

		return walkImgFs(layers, out)
	},
}

func positionalArgs(args, defaults []string) (defaulted []string) {
	for i := range defaults {
		if i < len(args) {
			defaulted = append(defaulted, args[i])
		} else {
			defaulted = append(defaulted, defaults[i])
		}
	}
	return
}

func walkImgFs(layers []containerregistry.Layer, w io.Writer) error {
	whiteouts := make(map[string]bool)

	tw := tar.NewWriter(w)
	defer tw.Close()

	for layerIdx, layer := range layers {
		ul, err := layer.Uncompressed()
		if err != nil {
			return fmt.Errorf("couldn't decompress layer[%d]: %v", layerIdx, err)
		}
		defer ul.Close()

		tarReader := tar.NewReader(ul)
		for {
			hdr, err := tarReader.Next()
			if err == io.EOF {
				break // End of archive
			}
			if err != nil {
				return fmt.Errorf("couldn't read next file in layer[%d]: %v", layerIdx, err)
			}

			if strings.HasPrefix(path.Base(hdr.FileInfo().Name()), WhiteoutPrefix) {
				whiteouts[hdr.Name] = true
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			if hdr.FileInfo().Size() > 0 {
				if _, err := io.Copy(tw, tarReader); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(img2fs)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// playLogCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// playLogCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
