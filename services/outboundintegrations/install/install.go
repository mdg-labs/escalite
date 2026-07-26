// Package install registers built-in outbound integration plugins.
package install

import (
	_ "github.com/mdg-labs/escalite/services/outboundintegrations/jira"
	_ "github.com/mdg-labs/escalite/services/outboundintegrations/servicenow"
)
