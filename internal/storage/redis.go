package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/company/version-service/internal/models"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

const (
	versionKeyPrefix = "version:"
	allVersionsKey   = "versions:all"
	defaultTTL       = 24 * time.Hour
)

type RedisStorage struct {
	client *redis.Client
	logger *logrus.Logger
}

func NewRedisStorage(redisURL string, logger *logrus.Logger) (*RedisStorage, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStorage{
		client: client,
		logger: logger,
	}, nil
}

func (r *RedisStorage) GetVersion(ctx context.Context, appID string) (*models.AppVersion, error) {
	key := versionKeyPrefix + appID

	data, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		r.logger.WithError(err).WithField("app_id", appID).Error("Failed to get version from Redis")
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	var version models.AppVersion
	if err := json.Unmarshal([]byte(data), &version); err != nil {
		r.logger.WithError(err).WithField("app_id", appID).Error("Failed to unmarshal version")
		return nil, fmt.Errorf("failed to unmarshal version: %w", err)
	}

	return &version, nil
}

func (r *RedisStorage) SetVersion(ctx context.Context, appID string, version *models.AppVersion) error {
	key := versionKeyPrefix + appID

	data, err := json.Marshal(version)
	if err != nil {
		r.logger.WithError(err).WithField("app_id", appID).Error("Failed to marshal version")
		return fmt.Errorf("failed to marshal version: %w", err)
	}

	pipe := r.client.TxPipeline()
	pipe.Set(ctx, key, data, defaultTTL)
	pipe.SAdd(ctx, allVersionsKey, appID)
	pipe.Expire(ctx, allVersionsKey, defaultTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		r.logger.WithError(err).WithField("app_id", appID).Error("Failed to set version in Redis")
		return fmt.Errorf("failed to set version: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"app_id":  appID,
		"version": version.Current,
	}).Debug("Version cached in Redis")

	return nil
}

func (r *RedisStorage) ListVersions(ctx context.Context) (map[string]*models.AppVersion, error) {
	appIDs, err := r.client.SMembers(ctx, allVersionsKey).Result()
	if err != nil {
		r.logger.WithError(err).Error("Failed to list version keys")
		return nil, fmt.Errorf("failed to list version keys: %w", err)
	}

	if len(appIDs) == 0 {
		return make(map[string]*models.AppVersion), nil
	}

	keys := make([]string, len(appIDs))
	for i, appID := range appIDs {
		keys[i] = versionKeyPrefix + appID
	}

	values, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		r.logger.WithError(err).Error("Failed to get multiple versions")
		return nil, fmt.Errorf("failed to get versions: %w", err)
	}

	versions := make(map[string]*models.AppVersion)
	for i, val := range values {
		if val == nil {
			continue
		}

		var version models.AppVersion
		if err := json.Unmarshal([]byte(val.(string)), &version); err != nil {
			r.logger.WithError(err).WithField("app_id", appIDs[i]).Warn("Failed to unmarshal version")
			continue
		}
		versions[appIDs[i]] = &version
	}

	return versions, nil
}

func (r *RedisStorage) ListVersionsByProject(ctx context.Context, projectID string) (map[string]*models.AppVersion, error) {
	allVersions, err := r.ListVersions(ctx)
	if err != nil {
		return nil, err
	}

	projectVersions := make(map[string]*models.AppVersion)
	for appID, version := range allVersions {
		if strings.HasPrefix(appID, projectID+"-") {
			projectVersions[appID] = version
		}
	}

	return projectVersions, nil
}

func (r *RedisStorage) DeleteVersion(ctx context.Context, appID string) error {
	key := versionKeyPrefix + appID

	pipe := r.client.TxPipeline()
	pipe.Del(ctx, key)
	pipe.SRem(ctx, allVersionsKey, appID)

	if _, err := pipe.Exec(ctx); err != nil {
		r.logger.WithError(err).WithField("app_id", appID).Error("Failed to delete version from Redis")
		return fmt.Errorf("failed to delete version: %w", err)
	}

	r.logger.WithField("app_id", appID).Debug("Version deleted from Redis")
	return nil
}

func (r *RedisStorage) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisStorage) RebuildCache(ctx context.Context, versions map[string]*models.AppVersion) error {
	pipe := r.client.TxPipeline()

	pipe.Del(ctx, allVersionsKey)

	for appID, version := range versions {
		key := versionKeyPrefix + appID
		data, err := json.Marshal(version)
		if err != nil {
			r.logger.WithError(err).WithField("app_id", appID).Warn("Failed to marshal version for cache rebuild")
			continue
		}
		pipe.Set(ctx, key, data, defaultTTL)
		pipe.SAdd(ctx, allVersionsKey, appID)
	}

	pipe.Expire(ctx, allVersionsKey, defaultTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		r.logger.WithError(err).Error("Failed to rebuild Redis cache")
		return fmt.Errorf("failed to rebuild cache: %w", err)
	}

	r.logger.WithField("count", len(versions)).Info("Redis cache rebuilt")
	return nil
}

func (r *RedisStorage) Close() error {
	return r.client.Close()
}