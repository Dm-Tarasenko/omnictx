# cloud-selection-cli

## REMOVED Requirements

### Requirement: AWS has no persistent switch — print the session hint
**Reason**: `omnictx cloud aws <profile>` is a real switch now (see the
`aws-profile-switch-cli` capability added by this change): it validates the
profile offline and persists `aws_profile:` to omnictx's own config, applied
by hook-running shells. The hint-and-exit-2 path is deleted.
