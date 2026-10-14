package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	configCmd.AddCommand(configSetCmd)
	addConfigFlags(configSetCmd.Flags())
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Updates the configuration",
	Long: `Updates the configuration. Set the flags for the options
you want to change. Other options will remain unchanged.`,
	Args: cobra.NoArgs,
	RunE: withStore(func(cmd *cobra.Command, _ []string, st *store) error {
		flags := cmd.Flags()

		// Read existing config
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

		// Get updated config
		auther, err = getSettings(flags, set, ser, auther, false)
		if err != nil {
			return err
		}

		// Save updated config
		err = st.SaveAuther(auther)
		if err != nil {
			return err
		}

		err = st.SaveSettings(set)
		if err != nil {
			return err
		}

		err = st.SaveServerConfig(ser)
		if err != nil {
			return err
		}

		return printSettings(ser, set, auther)
	}, storeOptions{}),
}
