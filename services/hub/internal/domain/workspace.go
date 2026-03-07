package domain

type WorkspaceID string

type Workspace struct {
	ID             WorkspaceID
	Name           string
	Mode           string
	HubURL         *string
	AuthMode       string
	LoginDisabled  bool
	IsDefaultLocal bool
	CreatedAt      string
}
