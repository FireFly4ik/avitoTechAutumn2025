package gorm

import (
	"avitoTechAutumn2025/internal/metrics"
	"avitoTechAutumn2025/internal/storage"
	"context"
	"time"

	"gorm.io/gorm"
)

// txManager реализует storage.TxManager для GORM
type txManager struct {
	db *gorm.DB
}

// NewTxManager создаёт новый менеджер транзакций для GORM.
// Не запускает фоновых горутин — для метрик используйте metrics.NewReconciler отдельно.
func NewTxManager(db *gorm.DB) storage.TxManager {
	return &txManager{db: db}
}

// Do выполняет функцию внутри транзакции с автоматическим commit/rollback.
// Используется для операций записи.
func (tm *txManager) Do(ctx context.Context, fn func(ctx context.Context, tx storage.Tx) error) error {
	start := time.Now()

	err := tm.db.WithContext(ctx).Transaction(func(gormTx *gorm.DB) error {
		tx := newTransaction(gormTx)

		if err := fn(ctx, tx); err != nil {
			metrics.DBTransactionTotal.WithLabelValues("error").Inc()
			return err
		}

		metrics.DBTransactionTotal.WithLabelValues("success").Inc()
		return nil
	})

	metrics.DBTransactionDuration.Observe(time.Since(start).Seconds())

	return err
}

// DoRead выполняет функцию без явной транзакции.
// Используется для read-only операций, чтобы избежать лишнего BEGIN/COMMIT overhead.
func (tm *txManager) DoRead(ctx context.Context, fn func(ctx context.Context, tx storage.Tx) error) error {
	tx := newTransaction(tm.db.WithContext(ctx))
	return fn(ctx, tx)
}

// transaction — обёртка над gorm.DB, реализует storage.Tx.
// Репозитории создаются лениво и кэшируются на время жизни транзакции.
type transaction struct {
	db     *gorm.DB
	prRepo storage.PullRequestRepository
	uRepo  storage.UserRepository
	tRepo  storage.TeamRepository
}

func newTransaction(db *gorm.DB) *transaction {
	return &transaction{db: db}
}

// PullRequestRepo возвращает репозиторий PR (кэшируется)
func (t *transaction) PullRequestRepo() storage.PullRequestRepository {
	if t.prRepo == nil {
		t.prRepo = NewPullRequestRepository(t.db)
	}
	return t.prRepo
}

// UserRepo возвращает репозиторий пользователей (кэшируется)
func (t *transaction) UserRepo() storage.UserRepository {
	if t.uRepo == nil {
		t.uRepo = NewUserRepository(t.db)
	}
	return t.uRepo
}

// TeamRepo возвращает репозиторий команд (кэшируется)
func (t *transaction) TeamRepo() storage.TeamRepository {
	if t.tRepo == nil {
		t.tRepo = NewTeamRepository(t.db)
	}
	return t.tRepo
}
