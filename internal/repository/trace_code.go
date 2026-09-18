package repository

import (
	"cc-052/internal/model"
	"time"

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

	stmt, err := tx.Preparex(`INSERT INTO trace_code (batch_id, code, seq, package_unit_code)
	                          VALUES ($1, $2, $3, $4) ON CONFLICT (code) DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, tc := range codes {
		if _, err := stmt.Exec(tc.BatchID, tc.Code, tc.Seq, tc.PackageUnitCode); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// traceCodeWithUnit 用于 JOIN 包装单位字典的查询
type traceCodeWithUnit struct {
	model.TraceCode
	PackageUnitName string `db:"package_unit_name"`
}

// GetByCode 按溯源码取回码记录及其包装单位（一条码↔一包装单位）
func (r *TraceCodeRepo) GetByCode(code string) (*model.TraceCode, *model.PackageUnit, error) {
	var row traceCodeWithUnit
	query := `SELECT tc.id, tc.batch_id, tc.code, tc.seq, tc.package_unit_code,
	                 tc.printed_at, tc.first_scanned_at, tc.first_scan_region, tc.created_at,
	                 pu.name AS package_unit_name
	          FROM trace_code tc
	          JOIN package_unit pu ON pu.code = tc.package_unit_code
	          WHERE tc.code = $1`
	if err := r.db.Get(&row, query, code); err != nil {
		return nil, nil, err
	}
	tc := row.TraceCode
	unit := &model.PackageUnit{Code: tc.PackageUnitCode, Name: row.PackageUnitName}
	return &tc, unit, nil
}

// RecordScan 原子记录一次扫码：
//   - 用条件 UPDATE（first_scanned_at IS NULL）抢占「首次扫码」，并发下也只会有一个请求判定为首次；
//   - 无论首次还是再次，都在 trace_scan 留痕，并返回含本次在内的累计扫码次数；
//   - 一并返回首次扫码时间与地区（再次扫码时供防伪提示使用）。
//
// region 为扫码端显式上报的行政区划码；未知（扫码端未授权定位）时传 nil，绝不用请求 IP 推断。
func (r *TraceCodeRepo) RecordScan(id int64, region *string) (
	isFirst bool, scanCount int, firstScannedAt time.Time, firstRegion *string, err error,
) {
	tx, err := r.db.Beginx()
	if err != nil {
		return false, 0, time.Time{}, nil, err
	}
	defer tx.Rollback()

	// 只有仍未记录首次扫码的码才会命中这一行更新，行锁保证并发互斥
	res, err := tx.Exec(`UPDATE trace_code
	                     SET first_scanned_at = NOW(), first_scan_region = $2
	                     WHERE id = $1 AND first_scanned_at IS NULL`, id, region)
	if err != nil {
		return false, 0, time.Time{}, nil, err
	}
	if n, _ := res.RowsAffected(); n == 1 {
		isFirst = true
	}

	if _, err := tx.Exec(`INSERT INTO trace_scan (trace_code_id, region_code) VALUES ($1, $2)`,
		id, region); err != nil {
		return false, 0, time.Time{}, nil, err
	}

	if err := tx.Get(&scanCount,
		`SELECT COUNT(*) FROM trace_scan WHERE trace_code_id = $1`, id); err != nil {
		return false, 0, time.Time{}, nil, err
	}

	if err := tx.QueryRowx(
		`SELECT first_scanned_at, first_scan_region FROM trace_code WHERE id = $1`, id,
	).Scan(&firstScannedAt, &firstRegion); err != nil {
		return false, 0, time.Time{}, nil, err
	}

	if err := tx.Commit(); err != nil {
		return false, 0, time.Time{}, nil, err
	}
	return isFirst, scanCount, firstScannedAt, firstRegion, nil
}

// GetPackageUnit 按 code 查包装单位，不存在返回 sql.ErrNoRows
func (r *TraceCodeRepo) GetPackageUnit(code string) (*model.PackageUnit, error) {
	var u model.PackageUnit
	query := `SELECT code, name, sort_order FROM package_unit WHERE code = $1`
	if err := r.db.Get(&u, query, code); err != nil {
		return nil, err
	}
	return &u, nil
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
