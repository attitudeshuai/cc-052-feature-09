package repository

import (
	"cc-052/internal/model"

	"github.com/jmoiron/sqlx"
)

type TraceCodeRepo struct {
	db *sqlx.DB
}

func NewTraceCodeRepo(db *sqlx.DB) *TraceCodeRepo {
	return &TraceCodeRepo{db: db}
}

func (r *TraceCodeRepo) BatchInsert(codes []model.TraceCode) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Preparex(`INSERT INTO trace_code (batch_id, code, seq) VALUES ($1, $2, $3) ON CONFLICT (code) DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, tc := range codes {
		if _, err := stmt.Exec(tc.BatchID, tc.Code, tc.Seq); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// RecordScan 原子地记录一次扫码并返回更新后的码记录：
// scan_count 恒加一；首扫时间与首扫地区只在首次扫码时写入，之后不再覆盖。
// region 为 nil 表示地区未知（服务端无法解析时绝不采信请求自报的来源）。
func (r *TraceCodeRepo) RecordScan(code string, region *string) (*model.TraceCode, error) {
	var tc model.TraceCode
	query := `UPDATE trace_code
	          SET scan_count = scan_count + 1,
	              first_scanned_at = COALESCE(first_scanned_at, NOW()),
	              first_scan_region = CASE WHEN first_scanned_at IS NULL THEN $2 ELSE first_scan_region END
	          WHERE code = $1
	          RETURNING id, batch_id, code, seq, printed_at, first_scanned_at, first_scan_region, scan_count, created_at`
	if err := r.db.Get(&tc, query, code, region); err != nil {
		return nil, err
	}
	return &tc, nil
}

func (r *TraceCodeRepo) GetMaxSeqByBatch(batchID int64) (int, error) {
	var maxSeq int
	query := `SELECT COALESCE(MAX(seq), 0) FROM trace_code WHERE batch_id = $1`
	if err := r.db.Get(&maxSeq, query, batchID); err != nil {
		return 0, err
	}
	return maxSeq, nil
}

func (r *TraceCodeRepo) CountByBatch(batchID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM trace_code WHERE batch_id = $1`
	if err := r.db.Get(&count, query, batchID); err != nil {
		return 0, err
	}
	return count, nil
}
