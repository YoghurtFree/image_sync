# image-sync Specification

## Purpose
TBD - created by archiving change image-sync-service. Update Purpose after archive.
## Requirements
### Requirement: Submit sync task
The system SHALL allow users to submit an image sync task specifying source registry name, target registry name, image name, tag, and priority. The system MUST return a task ID immediately. Registry credentials MUST be resolved from a config file loaded at startup.

#### Scenario: Successful submission
- **WHEN** user POST /api/v1/sync with {src, dst, image, tag, priority}
- **THEN** system enqueues task to Asynq and returns 202 with {task_id}

#### Scenario: Source registry not configured
- **WHEN** user submits a sync task with a source registry name not in config
- **THEN** system returns 400 Bad Request

#### Scenario: Target registry not configured
- **WHEN** user submits a sync task with a target registry name not in config
- **THEN** system returns 400 Bad Request

### Requirement: Priority queue ordering
The system SHALL process tasks by priority: high tasks MUST be processed before normal, normal before low.

#### Scenario: High priority jumps queue
- **WHEN** a low-priority task is pending and a high-priority task is submitted
- **THEN** the high-priority task MUST be processed first

### Requirement: Query task status
The system SHALL allow users to query the status of a sync task by task ID via Asynq Inspector.

#### Scenario: Task in progress
- **WHEN** user GET /api/v1/tasks/:id and task is running
- **THEN** system returns 200 with {status: "processing"}

#### Scenario: Task completed
- **WHEN** user GET /api/v1/tasks/:id and task succeeded
- **THEN** system returns 200 with {status: "completed", completed_at}

#### Scenario: Task failed
- **WHEN** user GET /api/v1/tasks/:id and task failed
- **THEN** system returns 200 with {status: "failed", error}

#### Scenario: Task not found
- **WHEN** user queries a non-existent task
- **THEN** system returns 404 Not Found

### Requirement: Image copy execution
The system SHALL use go-containerregistry to pull the image from the source registry and push to the target registry. The system MUST support parallel layer transfer.

#### Scenario: Successful copy
- **WHEN** a sync task executes and both registries are reachable
- **THEN** system copies all layers and manifest, marks task as completed

#### Scenario: Source registry unreachable
- **WHEN** a sync task executes and source registry is unreachable
- **THEN** system retries up to 3 times, then marks task as failed with error

#### Scenario: Target registry unreachable
- **WHEN** a sync task executes and target registry is unreachable
- **THEN** system retries up to 3 times, then marks task as failed with error

### Requirement: Multi-instance deployment
The system SHALL support deploying multiple worker instances. All task state MUST be managed by Asynq so no task is lost on instance failure.

#### Scenario: Worker instance restarts
- **WHEN** a worker instance crashes while processing a task
- **THEN** Asynq MUST re-queue the task for another worker to pick up

