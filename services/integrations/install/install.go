// Package install registers built-in inbound integration plugins.
package install

import (
	_ "github.com/mdg-labs/escalite/services/integrations/alertmanager"
	_ "github.com/mdg-labs/escalite/services/integrations/emailtoalert"
	_ "github.com/mdg-labs/escalite/services/integrations/genericrest"
	_ "github.com/mdg-labs/escalite/services/integrations/genericwebhook"
)
