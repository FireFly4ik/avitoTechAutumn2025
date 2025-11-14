package gorm

import (
	"avitoTechAutumn2025/internal/config"
	"context"

	"gorm.io/gorm"

	"avitoTechAutumn2025/internal/storage"
)

// txManager реализует storage.TxManager для GORM
type txManager struct {
	db *gorm.DB
}

// NewTxManager создаёт новый менеджер транзакций для GORM
func NewTxManager(envConf *config.Config) (storage.TxManager, error) {
	db, err := ConnectDB(envConf)
	if err != nil {
		return nil, err
	}
	return &txManager{db: db}, nil
}

// Do выполняет функцию внутри транзации с автоматическим commit/rollback
func (tm *txManager) Do(ctx context.Context, fn func(ctx context.Context, tx storage.Tx) error) error {
	return tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txWrapper := &transaction{
			db: tx,
		}

		err := fn(ctx, txWrapper)
		if err != nil {
			// GORM автоматически сделает ROLLBACK
			return err
		}

		// GORM автоматически сделает COMMIT
		return nil
	})
}

// transaction - обёртка над gorm.DB, реализует storage.Tx
type transaction struct {
	db *gorm.DB
}

// PullRequestRepo возвращает репозиторий PR в рамках транзакции
func (t *transaction) PullRequestRepo() storage.PullRequestRepository {
	return NewPullRequestRepository(t.db)
}

// UserRepo возвращает репозиторий пользователей в рамках транзакции
func (t *transaction) UserRepo() storage.UserRepository {
	return NewUserRepository(t.db)
}

// TeamRepo возвращает репозиторий команд в рамках транзакции
func (t *transaction) TeamRepo() storage.TeamRepository {
	return NewTeamRepository(t.db)
}

// Commit не нужен, так как GORM автоматически коммитит
func (t *transaction) Commit() error {
	return nil
}

// Rollback не нужен, так как GORM автоматически откатывает при ошибке
func (t *transaction) Rollback() error {
	return nil
}
