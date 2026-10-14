package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	configCmd.AddCommand(configCatCmd)
}

var configCatCmd = &cobra.Command{
	Use:   "cat",
	Short: "Prints the configuration",
	Long:  `Prints the configuration.`,
	Args:  cobra.NoArgs,
	RunE: withStore(func(_ *cobra.Command, _ []string, st *store) error {
		set, err := st.GetSettings()
		if err != nil {
			return err
		}
		ser, err := st.GetServerConfig()
		if err != nil {
			return err
		}
		auther, err := st.GetAuther(set.AuthMethod)
		if err != nil {
			return err
		}
		return printSettings(ser, set, auther)
	}, storeOptions{}),
}
