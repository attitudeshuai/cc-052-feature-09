package model

import "time"

type TraceCode struct {
	ID              int64      `db:"id" json:"id"`
	BatchID         int64      `db:"batch_id" json:"batch_id"`
	Code            string     `db:"code" json:"code"`
	Seq             int        `db:"seq" json:"seq"`
	PrintedAt       *time.Time `db:"printed_at" json:"printed_at,omitempty"`
	FirstScannedAt  *time.Time `db:"first_scanned_at" json:"first_scanned_at,omitempty"`
	FirstScanRegion *string    `db:"first_scan_region" json:"first_scan_region,omitempty"`
	ScanCount       int        `db:"scan_count" json:"scan_count"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
}

type GenerateCodeRequest struct {
	Count int `json:"count" binding:"required,min=1,max=1000"`
}

// TraceResponse 是公开扫码页的返回体，只保留核对真伪必需的内容：
// 不出现操作人、用量、施药次数等涉及个人信息与生产细节的数据。
type TraceResponse struct {
	Code            string                `json:"code"`
	Batch           *TraceBatchInfo       `json:"batch"`
	Farm            *TraceFarmInfo        `json:"farm"`
	Inspection      *TraceInspectionInfo  `json:"inspection,omitempty"`
	PackageUnit     *TracePackageUnitInfo `json:"package_unit"`
	FirstScan       bool                  `json:"first_scan"`
	ScanCount       int                   `json:"scan_count"`
	FirstScannedAt  string                `json:"first_scanned_at,omitempty"`
	FirstScanRegion string                `json:"first_scan_region,omitempty"`
}

type TraceBatchInfo struct {
	CropID      string `json:"crop_id"`
	SowingDate  string `json:"sowing_date"`
	HarvestDate string `json:"harvest_date,omitempty"`
}

type TraceFarmInfo struct {
	Name       string `json:"name"`
	RegionCode string `json:"region_code"`
}

// TracePackageUnitInfo 描述该码与包装单位的对应关系：批次内第几件 / 共几件。
type TracePackageUnitInfo struct {
	Seq   int `json:"seq"`
	Total int `json:"total"`
}

type TraceInspectionInfo struct {
	Lab       string           `json:"lab"`
	SampledAt string           `json:"sampled_at"`
	Result    InspectionResult `json:"result"`
}
