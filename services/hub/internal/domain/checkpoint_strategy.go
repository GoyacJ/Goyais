package domain

type CheckpointStrategyKind string

const (
	CheckpointStrategyGitCommit    CheckpointStrategyKind = "git_commit"
	CheckpointStrategyFileSnapshot CheckpointStrategyKind = "file_snapshot"
)

func ResolveCheckpointStrategy(kind CheckpointProjectKind) CheckpointStrategyKind {
	if kind == CheckpointProjectKindGit {
		return CheckpointStrategyGitCommit
	}
	return CheckpointStrategyFileSnapshot
}
