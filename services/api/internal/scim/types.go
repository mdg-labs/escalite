package scim

const (
	ContentType = "application/scim+json"
	SchemaUser  = "urn:ietf:params:scim:schemas:core:2.0:User"
	SchemaGroup = "urn:ietf:params:scim:schemas:core:2.0:Group"
	SchemaError = "urn:ietf:params:scim:api:messages:2.0:Error"
	SchemaList  = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	SchemaPatch = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
)

// Name represents a SCIM name attribute.
type Name struct {
	Formatted string `json:"formatted,omitempty"`
}

// Email represents a SCIM email attribute.
type Email struct {
	Value   string `json:"value"`
	Primary bool   `json:"primary,omitempty"`
}

// Member represents a SCIM group member reference.
type Member struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
}

// User is the SCIM 2.0 User resource.
type User struct {
	Schemas    []string `json:"schemas"`
	ID         string   `json:"id,omitempty"`
	ExternalID string   `json:"externalId,omitempty"`
	UserName   string   `json:"userName"`
	Name       *Name    `json:"name,omitempty"`
	Emails     []Email  `json:"emails,omitempty"`
	Active     bool     `json:"active"`
}

// Group is the SCIM 2.0 Group resource.
type Group struct {
	Schemas     []string `json:"schemas"`
	ID          string   `json:"id,omitempty"`
	ExternalID  string   `json:"externalId,omitempty"`
	DisplayName string   `json:"displayName"`
	Members     []Member `json:"members,omitempty"`
}

// ListResponse is a SCIM list response envelope.
type ListResponse struct {
	Schemas      []string `json:"schemas"`
	TotalResults int      `json:"totalResults"`
	StartIndex   int      `json:"startIndex"`
	ItemsPerPage int      `json:"itemsPerPage"`
	Resources    any      `json:"Resources"`
}

// PatchRequest is a SCIM PATCH request body.
type PatchRequest struct {
	Schemas    []string    `json:"schemas"`
	Operations []PatchOperation `json:"Operations"`
}

// PatchOperation is a single SCIM PATCH operation.
type PatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path,omitempty"`
	Value any    `json:"value,omitempty"`
}

// ErrorResponse is a SCIM error payload.
type ErrorResponse struct {
	Schemas  []string `json:"schemas"`
	Detail   string   `json:"detail"`
	Status   string   `json:"status"`
	ScimType string   `json:"scimType,omitempty"`
}

// ServiceProviderConfig is the SCIM service provider config resource.
type ServiceProviderConfig struct {
	Schemas    []string `json:"schemas"`
	Patch      map[string]bool `json:"patch"`
	Bulk       map[string]any `json:"bulk"`
	Filter     map[string]any `json:"filter"`
	ChangePass map[string]bool `json:"changePassword"`
	Sort       map[string]bool `json:"sort"`
	ETag       map[string]bool `json:"etag"`
	AuthSchemes []map[string]string `json:"authenticationSchemes"`
}
