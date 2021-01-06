package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/njhale/cmstore"
)

var (
	getExample = `
	# get and print a blob to stdout
	%[1]s get 123abcdef
	# get and print a blob to a file
	%[1]s get 123abcdef -o doge.svg
`
)

// GetOptions provides the information required to get a blob on-cluster.
type GetOptions struct {
	genericclioptions.IOStreams

	configFlags *genericclioptions.ConfigFlags
	args        []string
	out         string
	id          string

	blobWriter  io.Writer
	client      client.Client
	partitioner cmstore.Partitioner
}

// NewGetOptions provides an instance of GetOptions with default values.
func NewGetOptions(streams genericclioptions.IOStreams) *GetOptions {
	return &GetOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

// NewCmdGet provides a cobra command wrapping GetOptions
func NewCmdGet(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewGetOptions(streams)

	cmd := &cobra.Command{
		Use:          "get [id] [flags]",
		Short:        "Get a blob from the cluster",
		Example:      fmt.Sprintf(getExample, "kubectl stash"),
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

	cmd.Flags().StringVarP(&o.out, "out", "o", o.out, "File to store stashed blob output")
	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

// Complete sets all information required for updating the current context
func (o *GetOptions) Complete(cmd *cobra.Command, args []string) error {
	if len(args) == 1 {
		o.id = args[0]
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	o.client, err = client.New(cfg, client.Options{})
	if err != nil {
		return err
	}

	// TODO(njhale): make partition size configurable
	// Default partition size is 10KB
	o.partitioner = cmstore.NewPartitioner(1024 * 10)

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *GetOptions) Validate() error {
	// TODO(njhale): Do a better job validating arguments and flags
	if n := len(o.args); n > 1 {
		return fmt.Errorf("expected one argument, got %d", n)
	}

	return nil
}

// Run getes the contents of a blob on the cluster for later retrieval.
func (o *GetOptions) Run() error {
	var writer io.Writer
	if len(o.out) > 0 {
		fmt.Printf("o.out: %s\n", o.out)
		// A file has been specified, write to it
		f, err := os.Create(o.out)
		if err != nil {
			return err
		}
		defer f.Close()
		writer = f
	} else {
		// No file has been specified, write to stdout
		writer = o.IOStreams.Out
	}

	stream := cmstore.NewStream(o.client, "default", o.id)
	data := new([]byte)
	if err := o.partitioner.Join(data, stream); err != nil {
		return err
	}

	_, err := writer.Write(*data)

	return err
}
