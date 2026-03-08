package domain

type ResourceEventType string

const (
	ResourceEventTypeCreated            ResourceEventType = "resource_config_created"
	ResourceEventTypeUpdated            ResourceEventType = "resource_config_updated"
	ResourceEventTypeDeleted            ResourceEventType = "resource_config_deleted"
	ResourceEventTypeSnapshotDeprecated ResourceEventType = "resource_snapshot_deprecated"
)

type ResourceEvent struct {
	EventID         string
	WorkspaceID     WorkspaceID
	Type            ResourceEventType
	ConfigID        string
	ConfigType      ResourceType
	ResourceVersion int
	SessionID       SessionID
	Timestamp       string
	Payload         map[string]any
}

type AffectedSessionResources struct {
	Session       SessionResourceState
	ProjectConfig ProjectResourceConfig
}

type DeletedResourcePlan struct {
	Session   SessionResourceState
	Snapshots []SessionResourceSnapshot
	Event     ResourceEvent
}
