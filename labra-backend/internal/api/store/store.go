package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	db *sql.DB
}

type rowScanner interface {
	Scan(dest ...any) error
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func scanDeployment(scanner rowScanner, dep *Deployment) error {
	var retryableInt int
	if err := scanner.Scan(
		&dep.ID,
		&dep.AppID,
		&dep.UserID,
		&dep.Status,
		&dep.TriggerType,
		&dep.CommitSHA,
		&dep.CommitMessage,
		&dep.CommitAuthor,
		&dep.Branch,
		&dep.SiteURL,
		&dep.FailureReason,
		&dep.FailureCategory,
		&retryableInt,
		&dep.CorrelationID,
		&dep.ReleaseID,
		&dep.CreatedAt,
		&dep.UpdatedAt,
		&dep.StartedAt,
		&dep.FinishedAt,
	); err != nil {
		return err
	}
	dep.Retryable = retryableInt == 1
	return nil
}

func scanDeploymentJob(scanner rowScanner, job *DeploymentJob) error {
	return scanner.Scan(
		&job.ID,
		&job.DeploymentID,
		&job.AppID,
		&job.UserID,
		&job.Status,
		&job.AttemptCount,
		&job.MaxAttempts,
		&job.NextAttemptAt,
		&job.LastError,
		&job.ErrorCategory,
		&job.ClaimedBy,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.StartedAt,
		&job.FinishedAt,
	)
}

func (s *Store) CreateApp(ctx context.Context, in CreateAppInput) (App, error) {
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO apps (
			user_id, name, repo_full_name, branch, build_type, output_dir, root_dir, site_url, auto_deploy_enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, unixepoch(), unixepoch())
		RETURNING id, user_id, name, repo_full_name, branch, build_type, output_dir, COALESCE(root_dir, ''), COALESCE(site_url, ''), auto_deploy_enabled, COALESCE(current_release_version_id, 0), created_at, updated_at
	`, in.UserID, in.Name, in.RepoFullName, in.Branch, in.BuildType, in.OutputDir, nullIfEmpty(in.RootDir), nullIfEmpty(in.SiteURL), boolToInt(in.AutoDeployEnabled))

	var app App
	var autoDeployInt int
	if err := row.Scan(&app.ID, &app.UserID, &app.Name, &app.RepoFullName, &app.Branch, &app.BuildType, &app.OutputDir, &app.RootDir, &app.SiteURL, &autoDeployInt, &app.CurrentReleaseID, &app.CreatedAt, &app.UpdatedAt); err != nil {
		return App{}, err
	}
	app.AutoDeployEnabled = autoDeployInt == 1
	return app, nil
}

func (s *Store) ListAppsByUser(ctx context.Context, userID int64) ([]App, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, name, repo_full_name, branch, build_type, output_dir, COALESCE(root_dir, ''), COALESCE(site_url, ''), auto_deploy_enabled, COALESCE(current_release_version_id, 0), created_at, updated_at
		FROM apps
		WHERE user_id = ?
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]App, 0)
	for rows.Next() {
		var app App
		var autoDeployInt int
		if err := rows.Scan(&app.ID, &app.UserID, &app.Name, &app.RepoFullName, &app.Branch, &app.BuildType, &app.OutputDir, &app.RootDir, &app.SiteURL, &autoDeployInt, &app.CurrentReleaseID, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, err
		}
		app.AutoDeployEnabled = autoDeployInt == 1
		out = append(out, app)
	}
	return out, rows.Err()
}

func (s *Store) GetAppByIDForUser(ctx context.Context, appID, userID int64) (App, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, repo_full_name, branch, build_type, output_dir, COALESCE(root_dir, ''), COALESCE(site_url, ''), auto_deploy_enabled, COALESCE(current_release_version_id, 0), created_at, updated_at
		FROM apps
		WHERE id = ? AND user_id = ?
	`, appID, userID)

	var app App
	var autoDeployInt int
	if err := row.Scan(&app.ID, &app.UserID, &app.Name, &app.RepoFullName, &app.Branch, &app.BuildType, &app.OutputDir, &app.RootDir, &app.SiteURL, &autoDeployInt, &app.CurrentReleaseID, &app.CreatedAt, &app.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return App{}, ErrNotFound
		}
		return App{}, err
	}
	app.AutoDeployEnabled = autoDeployInt == 1
	return app, nil
}

func (s *Store) UpdateAppForUser(ctx context.Context, appID, userID int64, in UpdateAppInput) (App, error) {
	row := s.db.QueryRowContext(ctx, `
		UPDATE apps
		SET name = ?, branch = ?, build_type = ?, output_dir = ?, root_dir = ?, site_url = ?, auto_deploy_enabled = ?, updated_at = unixepoch()
		WHERE id = ? AND user_id = ?
		RETURNING id, user_id, name, repo_full_name, branch, build_type, output_dir, COALESCE(root_dir, ''), COALESCE(site_url, ''), auto_deploy_enabled, COALESCE(current_release_version_id, 0), created_at, updated_at
	`, in.Name, in.Branch, in.BuildType, in.OutputDir, nullIfEmpty(in.RootDir), nullIfEmpty(in.SiteURL), boolToInt(in.AutoDeployEnabled), appID, userID)

	var app App
	var autoDeployInt int
	if err := row.Scan(&app.ID, &app.UserID, &app.Name, &app.RepoFullName, &app.Branch, &app.BuildType, &app.OutputDir, &app.RootDir, &app.SiteURL, &autoDeployInt, &app.CurrentReleaseID, &app.CreatedAt, &app.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return App{}, ErrNotFound
		}
		return App{}, err
	}
	app.AutoDeployEnabled = autoDeployInt == 1
	return app, nil
}

func (s *Store) ListAutoDeployAppsByRepo(ctx context.Context, repoFullName string) ([]App, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, name, repo_full_name, branch, build_type, output_dir, COALESCE(root_dir, ''), COALESCE(site_url, ''), auto_deploy_enabled, COALESCE(current_release_version_id, 0), created_at, updated_at
		FROM apps
		WHERE lower(repo_full_name) = lower(?) AND auto_deploy_enabled = 1
		ORDER BY id ASC
	`, repoFullName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]App, 0)
	for rows.Next() {
		var app App
		var autoDeployInt int
		if err := rows.Scan(&app.ID, &app.UserID, &app.Name, &app.RepoFullName, &app.Branch, &app.BuildType, &app.OutputDir, &app.RootDir, &app.SiteURL, &autoDeployInt, &app.CurrentReleaseID, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, err
		}
		app.AutoDeployEnabled = autoDeployInt == 1
		out = append(out, app)
	}
	return out, rows.Err()
}

func (s *Store) CreateDeployment(ctx context.Context, in CreateDeploymentInput) (Deployment, error) {
	var startedAt any
	var finishedAt any
	if in.StartedAt > 0 {
		startedAt = in.StartedAt
	}
	if in.FinishedAt > 0 {
		finishedAt = in.FinishedAt
	}

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO deployments (
			app_id, user_id, status, trigger_type, commit_sha, commit_message, commit_author, branch, site_url, failure_reason, failure_category, retryable, correlation_id, release_version_id,
			created_at, updated_at, started_at, finished_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, unixepoch(), unixepoch(), ?, ?)
		RETURNING id, app_id, user_id, status, trigger_type, COALESCE(commit_sha, ''), COALESCE(commit_message, ''), COALESCE(commit_author, ''),
			COALESCE(branch, ''), COALESCE(site_url, ''), COALESCE(failure_reason, ''), COALESCE(failure_category, ''), COALESCE(retryable, 0), COALESCE(correlation_id, ''), COALESCE(release_version_id, 0), created_at, updated_at,
			COALESCE(started_at, 0), COALESCE(finished_at, 0)
	`, in.AppID, in.UserID, in.Status, in.TriggerType, nullIfEmpty(in.CommitSHA), nullIfEmpty(in.CommitMessage), nullIfEmpty(in.CommitAuthor),
		nullIfEmpty(in.Branch), nullIfEmpty(in.SiteURL), nullIfEmpty(in.FailureReason), nullIfEmpty(in.FailureCategory), boolToInt(in.Retryable), nullIfEmpty(in.CorrelationID), nil, startedAt, finishedAt)

	var dep Deployment
	if err := scanDeployment(row, &dep); err != nil {
		return Deployment{}, err
	}
	return dep, nil
}

func (s *Store) GetDeploymentByIDForUser(ctx context.Context, deploymentID, userID int64) (Deployment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, app_id, user_id, status, trigger_type, COALESCE(commit_sha, ''), COALESCE(commit_message, ''), COALESCE(commit_author, ''),
			COALESCE(branch, ''), COALESCE(site_url, ''), COALESCE(failure_reason, ''), COALESCE(failure_category, ''), COALESCE(retryable, 0), COALESCE(correlation_id, ''), COALESCE(release_version_id, 0), created_at, updated_at,
			COALESCE(started_at, 0), COALESCE(finished_at, 0)
		FROM deployments
		WHERE id = ? AND user_id = ?
	`, deploymentID, userID)

	var dep Deployment
	if err := scanDeployment(row, &dep); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Deployment{}, ErrNotFound
		}
		return Deployment{}, err
	}
	return dep, nil
}

func (s *Store) ListDeploymentsByAppForUser(ctx context.Context, appID, userID int64) ([]Deployment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, app_id, user_id, status, trigger_type, COALESCE(commit_sha, ''), COALESCE(commit_message, ''), COALESCE(commit_author, ''),
			COALESCE(branch, ''), COALESCE(site_url, ''), COALESCE(failure_reason, ''), COALESCE(failure_category, ''), COALESCE(retryable, 0), COALESCE(correlation_id, ''), COALESCE(release_version_id, 0), created_at, updated_at,
			COALESCE(started_at, 0), COALESCE(finished_at, 0)
		FROM deployments
		WHERE app_id = ? AND user_id = ?
		ORDER BY created_at DESC
	`, appID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Deployment, 0)
	for rows.Next() {
		var dep Deployment
		if err := scanDeployment(rows, &dep); err != nil {
			return nil, err
		}
		out = append(out, dep)
	}
	return out, rows.Err()
}

func (s *Store) GetLatestDeploymentByAppForUser(ctx context.Context, appID, userID int64) (Deployment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, app_id, user_id, status, trigger_type, COALESCE(commit_sha, ''), COALESCE(commit_message, ''), COALESCE(commit_author, ''),
			COALESCE(branch, ''), COALESCE(site_url, ''), COALESCE(failure_reason, ''), COALESCE(failure_category, ''), COALESCE(retryable, 0), COALESCE(correlation_id, ''), COALESCE(release_version_id, 0), created_at, updated_at,
			COALESCE(started_at, 0), COALESCE(finished_at, 0)
		FROM deployments
		WHERE app_id = ? AND user_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, appID, userID)

	var dep Deployment
	if err := scanDeployment(row, &dep); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Deployment{}, ErrNotFound
		}
		return Deployment{}, err
	}
	return dep, nil
}

func (s *Store) GetLastSuccessfulDeploymentByAppForUser(ctx context.Context, appID, userID int64) (Deployment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, app_id, user_id, status, trigger_type, COALESCE(commit_sha, ''), COALESCE(commit_message, ''), COALESCE(commit_author, ''),
			COALESCE(branch, ''), COALESCE(site_url, ''), COALESCE(failure_reason, ''), COALESCE(failure_category, ''), COALESCE(retryable, 0), COALESCE(correlation_id, ''), COALESCE(release_version_id, 0), created_at, updated_at,
			COALESCE(started_at, 0), COALESCE(finished_at, 0)
		FROM deployments
		WHERE app_id = ? AND user_id = ? AND status = 'succeeded'
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, appID, userID)

	var dep Deployment
	if err := scanDeployment(row, &dep); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Deployment{}, ErrNotFound
		}
		return Deployment{}, err
	}
	return dep, nil
}

func (s *Store) UpdateDeploymentStatus(ctx context.Context, deploymentID int64, status, reason, siteURL string, startedAt, finishedAt int64) (Deployment, error) {
	var started any
	var finished any
	if startedAt > 0 {
		started = startedAt
	}
	if finishedAt > 0 {
		finished = finishedAt
	}

	row := s.db.QueryRowContext(ctx, `
		UPDATE deployments
		SET status = ?, failure_reason = ?, site_url = ?, updated_at = unixepoch(), started_at = COALESCE(?, started_at), finished_at = COALESCE(?, finished_at)
		WHERE id = ?
		RETURNING id, app_id, user_id, status, trigger_type, COALESCE(commit_sha, ''), COALESCE(commit_message, ''), COALESCE(commit_author, ''),
			COALESCE(branch, ''), COALESCE(site_url, ''), COALESCE(failure_reason, ''), COALESCE(failure_category, ''), COALESCE(retryable, 0), COALESCE(correlation_id, ''), COALESCE(release_version_id, 0), created_at, updated_at,
			COALESCE(started_at, 0), COALESCE(finished_at, 0)
	`, status, nullIfEmpty(reason), nullIfEmpty(siteURL), started, finished, deploymentID)

	var dep Deployment
	if err := scanDeployment(row, &dep); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Deployment{}, ErrNotFound
		}
		return Deployment{}, err
	}
	return dep, nil
}

func (s *Store) CreateDeploymentLog(ctx context.Context, deploymentID int64, level, message string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO deployment_logs (deployment_id, log_level, message, created_at)
		VALUES (?, ?, ?, unixepoch())
	`, deploymentID, level, message)
	return err
}

func (s *Store) ListDeploymentLogs(ctx context.Context, deploymentID int64) ([]DeploymentLog, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, deployment_id, log_level, message, created_at
		FROM deployment_logs
		WHERE deployment_id = ?
		ORDER BY created_at ASC, id ASC
	`, deploymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DeploymentLog, 0)
	for rows.Next() {
		var l DeploymentLog
		if err := rows.Scan(&l.ID, &l.DeploymentID, &l.LogLevel, &l.Message, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) QueryDeploymentLogsByAppForUser(ctx context.Context, appID, userID int64, query string, limit int) ([]LogQueryHit, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	filter := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT l.id, l.deployment_id, l.log_level, l.message, l.created_at, d.status, d.trigger_type, COALESCE(d.release_version_id, 0)
		FROM deployment_logs l
		JOIN deployments d ON d.id = l.deployment_id
		WHERE d.app_id = ? AND d.user_id = ? AND (
			lower(l.message) LIKE ? OR lower(l.log_level) LIKE ?
		)
		ORDER BY l.created_at DESC, l.id DESC
		LIMIT ?
	`, appID, userID, filter, filter, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]LogQueryHit, 0)
	for rows.Next() {
		var hit LogQueryHit
		if err := rows.Scan(&hit.LogID, &hit.DeploymentID, &hit.LogLevel, &hit.Message, &hit.CreatedAt, &hit.Status, &hit.TriggerType, &hit.ReleaseID); err != nil {
			return nil, err
		}
		out = append(out, hit)
	}
	return out, rows.Err()
}

func (s *Store) ListRecentDeploymentsByAppForUser(ctx context.Context, appID, userID int64, limit int) ([]Deployment, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, app_id, user_id, status, trigger_type, COALESCE(commit_sha, ''), COALESCE(commit_message, ''), COALESCE(commit_author, ''),
			COALESCE(branch, ''), COALESCE(site_url, ''), COALESCE(failure_reason, ''), COALESCE(failure_category, ''), COALESCE(retryable, 0), COALESCE(correlation_id, ''), COALESCE(release_version_id, 0), created_at, updated_at,
			COALESCE(started_at, 0), COALESCE(finished_at, 0)
		FROM deployments
		WHERE app_id = ? AND user_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`, appID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Deployment, 0)
	for rows.Next() {
		var dep Deployment
		if err := scanDeployment(rows, &dep); err != nil {
			return nil, err
		}
		out = append(out, dep)
	}
	return out, rows.Err()
}

func (s *Store) UpdateDeploymentOutcome(ctx context.Context, deploymentID int64, status, reason, category string, retryable bool, siteURL string, startedAt, finishedAt int64) (Deployment, error) {
	var started any
	var finished any
	if startedAt > 0 {
		started = startedAt
	}
	if finishedAt > 0 {
		finished = finishedAt
	}

	row := s.db.QueryRowContext(ctx, `
		UPDATE deployments
		SET status = ?,
			failure_reason = ?,
			failure_category = ?,
			retryable = ?,
			site_url = ?,
			updated_at = unixepoch(),
			started_at = COALESCE(?, started_at),
			finished_at = COALESCE(?, finished_at)
		WHERE id = ?
		RETURNING id, app_id, user_id, status, trigger_type, COALESCE(commit_sha, ''), COALESCE(commit_message, ''), COALESCE(commit_author, ''),
			COALESCE(branch, ''), COALESCE(site_url, ''), COALESCE(failure_reason, ''), COALESCE(failure_category, ''), COALESCE(retryable, 0), COALESCE(correlation_id, ''), COALESCE(release_version_id, 0),
			created_at, updated_at, COALESCE(started_at, 0), COALESCE(finished_at, 0)
	`, status, nullIfEmpty(reason), nullIfEmpty(category), boolToInt(retryable), nullIfEmpty(siteURL), started, finished, deploymentID)

	var dep Deployment
	if err := scanDeployment(row, &dep); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Deployment{}, ErrNotFound
		}
		return Deployment{}, err
	}
	return dep, nil
}

func (s *Store) CreateReleaseVersion(ctx context.Context, in CreateReleaseVersionInput) (ReleaseVersion, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReleaseVersion{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var nextVersion int64
	if err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version_number), 0) + 1
		FROM release_versions
		WHERE app_id = ?
	`, in.AppID).Scan(&nextVersion); err != nil {
		return ReleaseVersion{}, err
	}

	var out ReleaseVersion
	var retainedInt int
	if err = tx.QueryRowContext(ctx, `
		INSERT INTO release_versions (
			app_id, deployment_id, version_number, artifact_path, artifact_checksum, is_retained, created_at
		) VALUES (?, ?, ?, ?, ?, 1, unixepoch())
		RETURNING id, app_id, deployment_id, version_number, artifact_path, COALESCE(artifact_checksum, ''), is_retained, created_at
	`, in.AppID, in.DeploymentID, nextVersion, in.ArtifactPath, nullIfEmpty(in.ArtifactChecksum)).Scan(
		&out.ID,
		&out.AppID,
		&out.DeploymentID,
		&out.VersionNumber,
		&out.ArtifactPath,
		&out.ArtifactChecksum,
		&retainedInt,
		&out.CreatedAt,
	); err != nil {
		return ReleaseVersion{}, err
	}
	out.IsRetained = retainedInt == 1

	if _, err = tx.ExecContext(ctx, `
		UPDATE apps
		SET current_release_version_id = ?, updated_at = unixepoch()
		WHERE id = ?
	`, out.ID, in.AppID); err != nil {
		return ReleaseVersion{}, err
	}

	if _, err = tx.ExecContext(ctx, `
		UPDATE deployments
		SET release_version_id = ?, updated_at = unixepoch()
		WHERE id = ?
	`, out.ID, in.DeploymentID); err != nil {
		return ReleaseVersion{}, err
	}

	if err = tx.Commit(); err != nil {
		return ReleaseVersion{}, err
	}

	return out, nil
}

func (s *Store) GetReleaseVersionByIDForUser(ctx context.Context, appID, releaseID, userID int64) (ReleaseVersion, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT r.id, r.app_id, r.deployment_id, r.version_number, r.artifact_path, COALESCE(r.artifact_checksum, ''), r.is_retained, r.created_at
		FROM release_versions r
		JOIN apps a ON a.id = r.app_id
		WHERE r.id = ? AND r.app_id = ? AND a.user_id = ?
	`, releaseID, appID, userID)

	var out ReleaseVersion
	var retainedInt int
	if err := row.Scan(&out.ID, &out.AppID, &out.DeploymentID, &out.VersionNumber, &out.ArtifactPath, &out.ArtifactChecksum, &retainedInt, &out.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ReleaseVersion{}, ErrNotFound
		}
		return ReleaseVersion{}, err
	}
	out.IsRetained = retainedInt == 1
	return out, nil
}

func (s *Store) GetCurrentReleaseVersionByAppForUser(ctx context.Context, appID, userID int64) (ReleaseVersion, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT r.id, r.app_id, r.deployment_id, r.version_number, r.artifact_path, COALESCE(r.artifact_checksum, ''), r.is_retained, r.created_at
		FROM apps a
		JOIN release_versions r ON r.id = a.current_release_version_id
		WHERE a.id = ? AND a.user_id = ?
	`, appID, userID)

	var out ReleaseVersion
	var retainedInt int
	if err := row.Scan(&out.ID, &out.AppID, &out.DeploymentID, &out.VersionNumber, &out.ArtifactPath, &out.ArtifactChecksum, &retainedInt, &out.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ReleaseVersion{}, ErrNotFound
		}
		return ReleaseVersion{}, err
	}
	out.IsRetained = retainedInt == 1
	return out, nil
}

func (s *Store) GetPreviousReleaseVersionByAppForUser(ctx context.Context, appID, userID int64, currentReleaseID int64) (ReleaseVersion, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT r.id, r.app_id, r.deployment_id, r.version_number, r.artifact_path, COALESCE(r.artifact_checksum, ''), r.is_retained, r.created_at
		FROM release_versions r
		JOIN apps a ON a.id = r.app_id
		WHERE r.app_id = ? AND a.user_id = ? AND r.id != ?
		ORDER BY r.version_number DESC
		LIMIT 1
	`, appID, userID, currentReleaseID)

	var out ReleaseVersion
	var retainedInt int
	if err := row.Scan(&out.ID, &out.AppID, &out.DeploymentID, &out.VersionNumber, &out.ArtifactPath, &out.ArtifactChecksum, &retainedInt, &out.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ReleaseVersion{}, ErrNotFound
		}
		return ReleaseVersion{}, err
	}
	out.IsRetained = retainedInt == 1
	return out, nil
}

func (s *Store) ListReleaseVersionsByAppForUser(ctx context.Context, appID, userID int64, limit int) ([]ReleaseVersion, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.app_id, r.deployment_id, r.version_number, r.artifact_path, COALESCE(r.artifact_checksum, ''), r.is_retained, r.created_at
		FROM release_versions r
		JOIN apps a ON a.id = r.app_id
		WHERE r.app_id = ? AND a.user_id = ?
		ORDER BY r.version_number DESC
		LIMIT ?
	`, appID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ReleaseVersion, 0)
	for rows.Next() {
		var release ReleaseVersion
		var retainedInt int
		if err := rows.Scan(&release.ID, &release.AppID, &release.DeploymentID, &release.VersionNumber, &release.ArtifactPath, &release.ArtifactChecksum, &retainedInt, &release.CreatedAt); err != nil {
			return nil, err
		}
		release.IsRetained = retainedInt == 1
		out = append(out, release)
	}
	return out, rows.Err()
}

func (s *Store) ApplyReleaseRetentionPolicy(ctx context.Context, appID int64, keep int, currentReleaseID int64) error {
	if keep <= 0 {
		keep = 20
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id
		FROM release_versions
		WHERE app_id = ?
		ORDER BY version_number DESC
	`, appID)
	if err != nil {
		return err
	}
	defer rows.Close()

	keepSet := map[int64]struct{}{}
	count := 0
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		if count < keep || id == currentReleaseID {
			keepSet[id] = struct{}{}
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `
		UPDATE release_versions
		SET is_retained = 0
		WHERE app_id = ?
	`, appID); err != nil {
		return err
	}

	for id := range keepSet {
		if _, err = tx.ExecContext(ctx, `
			UPDATE release_versions
			SET is_retained = 1
			WHERE id = ? AND app_id = ?
		`, id, appID); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Store) SetCurrentReleaseVersionForAppForUser(ctx context.Context, appID, releaseID, userID int64) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE apps
		SET current_release_version_id = ?, updated_at = unixepoch()
		WHERE id = ?
			AND user_id = ?
			AND EXISTS (
				SELECT 1
				FROM release_versions r
				WHERE r.id = ? AND r.app_id = ?
			)
	`, releaseID, appID, userID, releaseID, appID)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) AttachReleaseToDeployment(ctx context.Context, deploymentID, releaseID int64) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE deployments
		SET release_version_id = ?, updated_at = unixepoch()
		WHERE id = ?
	`, releaseID, deploymentID)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CreateRollbackEvent(ctx context.Context, in CreateRollbackEventInput) (RollbackEvent, error) {
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO rollback_events (
			app_id, user_id, from_release_version_id, to_release_version_id, deployment_id, reason, created_at
		) VALUES (?, ?, ?, ?, ?, ?, unixepoch())
		RETURNING id, app_id, user_id, COALESCE(from_release_version_id, 0), to_release_version_id, deployment_id, COALESCE(reason, ''), created_at
	`, in.AppID, in.UserID, nullIfZero(in.FromReleaseID), in.ToReleaseID, in.DeploymentID, nullIfEmpty(in.Reason))

	var event RollbackEvent
	if err := row.Scan(&event.ID, &event.AppID, &event.UserID, &event.FromReleaseID, &event.ToReleaseID, &event.DeploymentID, &event.Reason, &event.CreatedAt); err != nil {
		return RollbackEvent{}, err
	}
	return event, nil
}

func (s *Store) ListRollbackEventsByAppForUser(ctx context.Context, appID, userID int64, limit int) ([]RollbackEvent, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT e.id, e.app_id, e.user_id, COALESCE(e.from_release_version_id, 0), e.to_release_version_id, e.deployment_id, COALESCE(e.reason, ''), e.created_at
		FROM rollback_events e
		JOIN apps a ON a.id = e.app_id
		WHERE e.app_id = ? AND a.user_id = ?
		ORDER BY e.created_at DESC, e.id DESC
		LIMIT ?
	`, appID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]RollbackEvent, 0)
	for rows.Next() {
		var event RollbackEvent
		if err := rows.Scan(&event.ID, &event.AppID, &event.UserID, &event.FromReleaseID, &event.ToReleaseID, &event.DeploymentID, &event.Reason, &event.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

func (s *Store) ListAppEnvVarsByAppForUser(ctx context.Context, appID, userID int64) ([]AppEnvVar, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT e.id, e.app_id, e.key, e.value, e.is_secret, e.created_at, e.updated_at
		FROM app_env_vars e
		JOIN apps a ON a.id = e.app_id
		WHERE e.app_id = ? AND a.user_id = ?
		ORDER BY e.key ASC
	`, appID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AppEnvVar, 0)
	for rows.Next() {
		var envVar AppEnvVar
		var isSecretInt int
		if err := rows.Scan(&envVar.ID, &envVar.AppID, &envVar.Key, &envVar.Value, &isSecretInt, &envVar.CreatedAt, &envVar.UpdatedAt); err != nil {
			return nil, err
		}
		envVar.IsSecret = isSecretInt == 1
		out = append(out, envVar)
	}
	return out, rows.Err()
}

func (s *Store) ListAppEnvVarsForApp(ctx context.Context, appID int64) ([]AppEnvVar, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, app_id, key, value, is_secret, created_at, updated_at
		FROM app_env_vars
		WHERE app_id = ?
		ORDER BY key ASC
	`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AppEnvVar, 0)
	for rows.Next() {
		var envVar AppEnvVar
		var isSecretInt int
		if err := rows.Scan(&envVar.ID, &envVar.AppID, &envVar.Key, &envVar.Value, &isSecretInt, &envVar.CreatedAt, &envVar.UpdatedAt); err != nil {
			return nil, err
		}
		envVar.IsSecret = isSecretInt == 1
		out = append(out, envVar)
	}
	return out, rows.Err()
}

func (s *Store) CreateAppEnvVar(ctx context.Context, appID, userID int64, in CreateAppEnvVarInput) (AppEnvVar, error) {
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO app_env_vars (app_id, key, value, is_secret, created_at, updated_at)
		SELECT a.id, ?, ?, ?, unixepoch(), unixepoch()
		FROM apps a
		WHERE a.id = ? AND a.user_id = ?
		RETURNING id, app_id, key, value, is_secret, created_at, updated_at
	`, in.Key, in.Value, boolToInt(in.IsSecret), appID, userID)

	var envVar AppEnvVar
	var isSecretInt int
	if err := row.Scan(&envVar.ID, &envVar.AppID, &envVar.Key, &envVar.Value, &isSecretInt, &envVar.CreatedAt, &envVar.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AppEnvVar{}, ErrNotFound
		}
		return AppEnvVar{}, err
	}
	envVar.IsSecret = isSecretInt == 1
	return envVar, nil
}

func (s *Store) GetAppEnvVarByIDForUser(ctx context.Context, appID, envVarID, userID int64) (AppEnvVar, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT e.id, e.app_id, e.key, e.value, e.is_secret, e.created_at, e.updated_at
		FROM app_env_vars e
		JOIN apps a ON a.id = e.app_id
		WHERE e.id = ? AND e.app_id = ? AND a.user_id = ?
	`, envVarID, appID, userID)

	var envVar AppEnvVar
	var isSecretInt int
	if err := row.Scan(&envVar.ID, &envVar.AppID, &envVar.Key, &envVar.Value, &isSecretInt, &envVar.CreatedAt, &envVar.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AppEnvVar{}, ErrNotFound
		}
		return AppEnvVar{}, err
	}
	envVar.IsSecret = isSecretInt == 1
	return envVar, nil
}

func (s *Store) UpdateAppEnvVarForUser(ctx context.Context, appID, envVarID, userID int64, in UpdateAppEnvVarInput) (AppEnvVar, error) {
	row := s.db.QueryRowContext(ctx, `
		UPDATE app_env_vars
		SET key = ?, value = ?, is_secret = ?, updated_at = unixepoch()
		WHERE id = ?
			AND app_id = ?
			AND EXISTS (
				SELECT 1
				FROM apps a
				WHERE a.id = ? AND a.user_id = ?
			)
		RETURNING id, app_id, key, value, is_secret, created_at, updated_at
	`, in.Key, in.Value, boolToInt(in.IsSecret), envVarID, appID, appID, userID)

	var envVar AppEnvVar
	var isSecretInt int
	if err := row.Scan(&envVar.ID, &envVar.AppID, &envVar.Key, &envVar.Value, &isSecretInt, &envVar.CreatedAt, &envVar.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AppEnvVar{}, ErrNotFound
		}
		return AppEnvVar{}, err
	}
	envVar.IsSecret = isSecretInt == 1
	return envVar, nil
}

func (s *Store) DeleteAppEnvVarForUser(ctx context.Context, appID, envVarID, userID int64) error {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM app_env_vars
		WHERE id = ?
			AND app_id = ?
			AND EXISTS (
				SELECT 1
				FROM apps a
				WHERE a.id = ? AND a.user_id = ?
			)
	`, envVarID, appID, appID, userID)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) RecordAppDeploymentOutcome(ctx context.Context, appID int64, status string, startedAt, finishedAt int64, triggerType string) error {
	succInc := int64(0)
	failInc := int64(0)
	lastSuccessAt := int64(0)
	lastFailureAt := int64(0)
	rollbackInc := int64(0)
	lastDeployAt := int64(0)
	duration := int64(0)

	if finishedAt > 0 {
		lastDeployAt = finishedAt
	}
	if startedAt > 0 && finishedAt >= startedAt {
		duration = finishedAt - startedAt
	}

	switch strings.TrimSpace(strings.ToLower(status)) {
	case "succeeded":
		succInc = 1
		lastSuccessAt = finishedAt
	case "failed":
		failInc = 1
		lastFailureAt = finishedAt
	default:
		return nil
	}

	if strings.TrimSpace(strings.ToLower(triggerType)) == "rollback" {
		rollbackInc = 1
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO app_health_metrics (
			app_id, success_count, failure_count, last_success_at, last_failure_at, last_deploy_at, total_duration_seconds, latest_duration_seconds, rollback_count, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, unixepoch())
		ON CONFLICT(app_id) DO UPDATE SET
			success_count = app_health_metrics.success_count + excluded.success_count,
			failure_count = app_health_metrics.failure_count + excluded.failure_count,
			last_success_at = CASE
				WHEN excluded.last_success_at > 0 THEN excluded.last_success_at
				ELSE app_health_metrics.last_success_at
			END,
			last_failure_at = CASE
				WHEN excluded.last_failure_at > 0 THEN excluded.last_failure_at
				ELSE app_health_metrics.last_failure_at
			END,
			last_deploy_at = CASE
				WHEN excluded.last_deploy_at > 0 THEN excluded.last_deploy_at
				ELSE app_health_metrics.last_deploy_at
			END,
			total_duration_seconds = app_health_metrics.total_duration_seconds + excluded.total_duration_seconds,
			latest_duration_seconds = excluded.latest_duration_seconds,
			rollback_count = app_health_metrics.rollback_count + excluded.rollback_count,
			updated_at = unixepoch()
	`, appID, succInc, failInc, lastSuccessAt, lastFailureAt, lastDeployAt, duration, duration, rollbackInc)
	return err
}

func (s *Store) GetAppHealthMetricsForUser(ctx context.Context, appID, userID int64) (AppHealthMetrics, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT a.id,
			COALESCE(m.success_count, 0),
			COALESCE(m.failure_count, 0),
			COALESCE(m.last_success_at, 0),
			COALESCE(m.last_failure_at, 0),
			COALESCE(m.last_deploy_at, 0),
			COALESCE(m.total_duration_seconds, 0),
			COALESCE(m.latest_duration_seconds, 0),
			COALESCE(m.rollback_count, 0),
			COALESCE(m.updated_at, a.updated_at)
		FROM apps a
		LEFT JOIN app_health_metrics m ON m.app_id = a.id
		WHERE a.id = ? AND a.user_id = ?
	`, appID, userID)

	var metrics AppHealthMetrics
	if err := row.Scan(
		&metrics.AppID,
		&metrics.SuccessCount,
		&metrics.FailureCount,
		&metrics.LastSuccessAt,
		&metrics.LastFailureAt,
		&metrics.LastDeployAt,
		&metrics.TotalDuration,
		&metrics.LatestDuration,
		&metrics.RollbackCount,
		&metrics.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AppHealthMetrics{}, ErrNotFound
		}
		return AppHealthMetrics{}, err
	}
	return metrics, nil
}

func (s *Store) ClaimWebhookDelivery(ctx context.Context, appID int64, deliveryID, eventType, commitSHA string) (bool, error) {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhook_deliveries (app_id, delivery_id, event_type, commit_sha, received_at)
		VALUES (?, ?, ?, ?, unixepoch())
	`, appID, deliveryID, eventType, nullIfEmpty(commitSHA))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique constraint failed") {
			return false, nil
		}
		return false, fmt.Errorf("insert webhook delivery: %w", err)
	}
	return true, nil
}

func (s *Store) CreateDeploymentJob(ctx context.Context, in CreateDeploymentJobInput) (DeploymentJob, error) {
	maxAttempts := in.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO deployment_jobs (
			deployment_id, app_id, user_id, status, attempt_count, max_attempts, next_attempt_at, created_at, updated_at
		) VALUES (?, ?, ?, 'queued', 0, ?, unixepoch(), unixepoch(), unixepoch())
		RETURNING id, deployment_id, app_id, user_id, status, attempt_count, max_attempts, next_attempt_at,
			COALESCE(last_error, ''), COALESCE(error_category, ''), COALESCE(claimed_by, ''),
			created_at, updated_at, COALESCE(started_at, 0), COALESCE(finished_at, 0)
	`, in.DeploymentID, in.AppID, in.UserID, maxAttempts)

	var job DeploymentJob
	if err := scanDeploymentJob(row, &job); err != nil {
		return DeploymentJob{}, err
	}
	return job, nil
}

func (s *Store) GetDeploymentJobByDeploymentID(ctx context.Context, deploymentID int64) (DeploymentJob, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, deployment_id, app_id, user_id, status, attempt_count, max_attempts, next_attempt_at,
			COALESCE(last_error, ''), COALESCE(error_category, ''), COALESCE(claimed_by, ''),
			created_at, updated_at, COALESCE(started_at, 0), COALESCE(finished_at, 0)
		FROM deployment_jobs
		WHERE deployment_id = ?
	`, deploymentID)

	var job DeploymentJob
	if err := scanDeploymentJob(row, &job); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DeploymentJob{}, ErrNotFound
		}
		return DeploymentJob{}, err
	}
	return job, nil
}

func (s *Store) ListActiveDeploymentJobsByAppForUser(ctx context.Context, appID, userID int64) ([]DeploymentJob, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT j.id, j.deployment_id, j.app_id, j.user_id, j.status, j.attempt_count, j.max_attempts, j.next_attempt_at,
			COALESCE(j.last_error, ''), COALESCE(j.error_category, ''), COALESCE(j.claimed_by, ''),
			j.created_at, j.updated_at, COALESCE(j.started_at, 0), COALESCE(j.finished_at, 0)
		FROM deployment_jobs j
		JOIN apps a ON a.id = j.app_id
		WHERE j.app_id = ? AND a.user_id = ? AND j.status IN ('queued', 'running', 'retrying')
		ORDER BY j.created_at ASC, j.id ASC
	`, appID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DeploymentJob, 0)
	for rows.Next() {
		var job DeploymentJob
		if err := scanDeploymentJob(rows, &job); err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	return out, rows.Err()
}

func (s *Store) ClaimNextRunnableDeploymentJob(ctx context.Context, workerID string) (DeploymentJob, error) {
	row := s.db.QueryRowContext(ctx, `
		UPDATE deployment_jobs
		SET status = 'running',
			attempt_count = attempt_count + 1,
			claimed_by = ?,
			updated_at = unixepoch(),
			started_at = unixepoch()
		WHERE id = (
			SELECT j.id
			FROM deployment_jobs j
			WHERE j.status IN ('queued', 'retrying')
				AND j.next_attempt_at <= unixepoch()
				AND j.attempt_count < j.max_attempts
				AND NOT EXISTS (
					SELECT 1
					FROM deployment_jobs running
					WHERE running.app_id = j.app_id
						AND running.status = 'running'
						AND running.id != j.id
				)
			ORDER BY j.next_attempt_at ASC, j.id ASC
			LIMIT 1
		)
		RETURNING id, deployment_id, app_id, user_id, status, attempt_count, max_attempts, next_attempt_at,
			COALESCE(last_error, ''), COALESCE(error_category, ''), COALESCE(claimed_by, ''),
			created_at, updated_at, COALESCE(started_at, 0), COALESCE(finished_at, 0)
	`, strings.TrimSpace(workerID))

	var job DeploymentJob
	if err := scanDeploymentJob(row, &job); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DeploymentJob{}, ErrNotFound
		}
		if strings.Contains(strings.ToLower(err.Error()), "unique constraint failed") {
			return DeploymentJob{}, ErrNotFound
		}
		return DeploymentJob{}, err
	}
	return job, nil
}

func (s *Store) MarkDeploymentJobRetry(ctx context.Context, jobID int64, nextAttemptAt int64, errMsg, category string) (DeploymentJob, error) {
	row := s.db.QueryRowContext(ctx, `
		UPDATE deployment_jobs
		SET status = 'retrying',
			next_attempt_at = ?,
			last_error = ?,
			error_category = ?,
			updated_at = unixepoch(),
			finished_at = NULL
		WHERE id = ?
		RETURNING id, deployment_id, app_id, user_id, status, attempt_count, max_attempts, next_attempt_at,
			COALESCE(last_error, ''), COALESCE(error_category, ''), COALESCE(claimed_by, ''),
			created_at, updated_at, COALESCE(started_at, 0), COALESCE(finished_at, 0)
	`, nextAttemptAt, nullIfEmpty(errMsg), nullIfEmpty(category), jobID)

	var job DeploymentJob
	if err := scanDeploymentJob(row, &job); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DeploymentJob{}, ErrNotFound
		}
		return DeploymentJob{}, err
	}
	return job, nil
}

func (s *Store) MarkDeploymentJobFailed(ctx context.Context, jobID int64, errMsg, category string) (DeploymentJob, error) {
	row := s.db.QueryRowContext(ctx, `
		UPDATE deployment_jobs
		SET status = 'failed',
			last_error = ?,
			error_category = ?,
			updated_at = unixepoch(),
			finished_at = unixepoch()
		WHERE id = ?
		RETURNING id, deployment_id, app_id, user_id, status, attempt_count, max_attempts, next_attempt_at,
			COALESCE(last_error, ''), COALESCE(error_category, ''), COALESCE(claimed_by, ''),
			created_at, updated_at, COALESCE(started_at, 0), COALESCE(finished_at, 0)
	`, nullIfEmpty(errMsg), nullIfEmpty(category), jobID)

	var job DeploymentJob
	if err := scanDeploymentJob(row, &job); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DeploymentJob{}, ErrNotFound
		}
		return DeploymentJob{}, err
	}
	return job, nil
}

func (s *Store) MarkDeploymentJobSucceeded(ctx context.Context, jobID int64) (DeploymentJob, error) {
	row := s.db.QueryRowContext(ctx, `
		UPDATE deployment_jobs
		SET status = 'succeeded',
			last_error = NULL,
			error_category = NULL,
			updated_at = unixepoch(),
			finished_at = unixepoch()
		WHERE id = ?
		RETURNING id, deployment_id, app_id, user_id, status, attempt_count, max_attempts, next_attempt_at,
			COALESCE(last_error, ''), COALESCE(error_category, ''), COALESCE(claimed_by, ''),
			created_at, updated_at, COALESCE(started_at, 0), COALESCE(finished_at, 0)
	`, jobID)

	var job DeploymentJob
	if err := scanDeploymentJob(row, &job); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DeploymentJob{}, ErrNotFound
		}
		return DeploymentJob{}, err
	}
	return job, nil
}

func (s *Store) CreateDeploymentRollbackPayload(ctx context.Context, deploymentID, fromReleaseID, targetReleaseID int64, reason string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO deployment_rollback_payloads (deployment_id, from_release_id, target_release_id, reason, created_at)
		VALUES (?, ?, ?, ?, unixepoch())
	`, deploymentID, fromReleaseID, targetReleaseID, nullIfEmpty(reason))
	return err
}

func (s *Store) GetDeploymentRollbackPayload(ctx context.Context, deploymentID int64) (DeploymentRollbackPayload, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT deployment_id, from_release_id, target_release_id, COALESCE(reason, ''), created_at
		FROM deployment_rollback_payloads
		WHERE deployment_id = ?
	`, deploymentID)

	var payload DeploymentRollbackPayload
	if err := row.Scan(&payload.DeploymentID, &payload.FromReleaseID, &payload.TargetReleaseID, &payload.Reason, &payload.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DeploymentRollbackPayload{}, ErrNotFound
		}
		return DeploymentRollbackPayload{}, err
	}
	return payload, nil
}

func nullIfEmpty(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func nullIfZero(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func UnixNow() int64 {
	return time.Now().Unix()
}
