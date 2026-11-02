package alpha

import "github.com/spf13/cobra"

// Alpha APIs accept the metadata currency identifier; a display symbol may not identify a token.
const currencyIDHelp = "Alpha currency ID (e.g. USDS02b5a8dd2f59a098), not a symbol; get the currency field from 'gate-cli cex alpha market currencies --format json' (omit --currency; paginate with --page)"

var Cmd = &cobra.Command{
	Use:   "alpha",
	Short: "Alpha token trading commands",
}
