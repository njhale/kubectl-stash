package cmd

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/njhale/cmstore"
)

var (
	stashExample = `
	# stash a blob from stdin
	cat doge.svg | %[1]s stash -
	# stash a blob from a file
	%[1]s stash doge.svg
`
)

type Stream interface {
	io.Reader
	io.Writer
}

// StashOptions provides the information required to stash a blob on-cluster.
type StashOptions struct {
	genericclioptions.IOStreams

	configFlags *genericclioptions.ConfigFlags
	args        []string
	blobReader  io.Reader
	client      client.Client
	partitioner cmstore.Partitioner
}

// NewStashOptions provides an instance of StashOptions with default values.
func NewStashOptions(streams genericclioptions.IOStreams) *StashOptions {
	return &StashOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

// NewCmdStash provides a cobra command wrapping StashOptions
func NewCmdStash(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewStashOptions(streams)

	cmd := &cobra.Command{
		Use:          "stash [file] [flags]",
		Short:        "Stash a blob on the cluster",
		Example:      fmt.Sprintf(stashExample, "kubectl"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	o.configFlags.AddFlags(cmd.Flags())

	cmd.AddCommand(NewCmdGet(streams))

	return cmd
}

// Complete sets all information required for updating the current context
func (o *StashOptions) Complete(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		// Input blob is a file if specified
		var err error
		if o.blobReader, err = os.Open(args[0]); err != nil {
			return err
		}
	} else {
		// Otherwise, take the blob from stdin
		o.blobReader = o.In
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	o.client, err = client.New(cfg, client.Options{})
	if err != nil {
		return err
	}

	o.partitioner = cmstore.NewPartitioner(1024 * 10)

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *StashOptions) Validate() error {
	// TODO(njhale): Do a better job validating arguments and flags
	if len(o.args) > 1 {
		return fmt.Errorf("either one or no arguments are allowed")
	}

	return nil
}

// Run stashes the contents of a blob on the cluster for later retrieval.
func (o *StashOptions) Run() error {
	var (
		buf bytes.Buffer
		err error
	)
	if _, err = buf.ReadFrom(o.blobReader); err != nil {
		return err
	}

	data := buf.Bytes()
	var hash string
	hash, err = safeHash64(buf.String())
	if err != nil {
		return err
	}
	id := hash

	stream := cmstore.NewStream(o.client, "default", id)
	if err = o.partitioner.Split(data, stream); err != nil {
		return err
	}

	_, err = o.IOStreams.Out.Write([]byte(id))

	return err
}

func safeHash64(s string) (string, error) {
	hasher := fnv.New64a()
	if _, err := hasher.Write([]byte(s)); err != nil {
		return "", nil
	}

	sum := hasher.Sum(nil)
	safe := strings.Map(func(r rune) rune {
		// Strip any \x00 control characters (SafeEndcodeString can't handle them)
		if r == 0 {
			return 'z'
		}
		return r
	}, rand.SafeEncodeString(string(sum)))

	fmt.Printf("safe: %s, len: %d\n", safe, len(safe))
	for _, r := range []rune(safe) {
		fmt.Printf("rune: %c\n", r)
	}

	return safe, nil
}
