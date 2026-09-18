package model

import "time"

type TraceCode struct {
	ID              int64      `db:"id" json:"id"`
	BatchID         int64      `db:"batch_id" json:"batch_id"`
	Code            string     `db:"code" json:"code"`
	Seq             int        `db:"seq" json:"seq"`
	PackageUnitCode string     `db:"package_unit_code" json:"package_unit_code"`
	PrintedAt       *time.Time `db:"printed_at" json:"printed_at,omitempty"`
	FirstScannedAt  *time.Time `db:"first_scanned_at" json:"first_scanned_at,omitempty"`
	FirstScanRegion *string    `db:"first_scan_region" json:"first_scan_region,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
}

// PackageUnit 包装单位字典（袋/盒/箱…）
type PackageUnit struct {
	Code      string `db:"code" json:"code"`
	Name      string `db:"name" json:"name"`
	SortOrder int    `db:"sort_order" json:"sort_order"`
}

type GenerateCodeRequest struct {
	Count           int    `json:"count" binding:"required,min=1,max=1000"`
	PackageUnitCode string `json:"package_unit" binding:"required"`
}

// TraceResponse 是公开扫码页的数据。
// 只包含核对真伪必需的批次/地块/检测/包装信息：
// 农事活动明细（操作人姓名、用量、施药次数等个人与生产信息）一律不下发。
type TraceResponse struct {
	Code        string               `json:"code"`
	PackageUnit *TracePackageInfo    `json:"package_unit"`
	Batch       *TraceBatchInfo      `json:"batch"`
	Farm        *TraceFarmInfo       `json:"farm"`
	Inspection  *TraceInspectionInfo `json:"inspection,omitempty"`
	// FirstScan: 本次扫码是否为该码的首次扫码（原子判定，并发安全）
	FirstScan bool `json:"first_scan"`
	// ScanCount: 截至本次扫码该码的累计扫码次数（1=首次，>1=再次扫码）
	ScanCount int `json:"scan_count"`
	// 首次扫码的防伪留痕信息（地区为扫码端上报的行政区划码，非请求来源 IP）
	FirstScannedAt  string `json:"first_scanned_at,omitempty"`
	FirstScanRegion string `json:"first_scan_region,omitempty"`
}

type TracePackageInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type TraceBatchInfo struct {
	CropID      string `json:"crop_id"`
	SowingDate  string `json:"sowing_date"`
	HarvestDate string `json:"harvest_date,omitempty"`
}

type TraceFarmInfo struct {
	Name       string `json:"name"`
	RegionCode string `json:"region_code"`
	PlotName   string `json:"plot_name"`
}

type TraceInspectionInfo struct {
	Lab       string           `json:"lab"`
	SampledAt string           `json:"sampled_at"`
	Result    InspectionResult `json:"result"`
}
