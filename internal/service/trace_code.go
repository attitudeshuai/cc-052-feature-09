package service

import (
	"cc-052/internal/model"
	"cc-052/internal/repository"
	"cc-052/pkg/tracecode"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrTraceCodeNotFound 表示溯源码不存在。
var ErrTraceCodeNotFound = errors.New("trace code not found")

type TraceCodeService struct {
	codeRepo       *repository.TraceCodeRepo
	batchRepo      *repository.BatchRepo
	inspectionRepo *repository.InspectionRepo
	plotRepo       *repository.PlotRepo
	farmRepo       *repository.FarmRepo
}

func NewTraceCodeService(
	codeRepo *repository.TraceCodeRepo,
	batchRepo *repository.BatchRepo,
	inspectionRepo *repository.InspectionRepo,
	plotRepo *repository.PlotRepo,
	farmRepo *repository.FarmRepo,
) *TraceCodeService {
	return &TraceCodeService{
		codeRepo:       codeRepo,
		batchRepo:      batchRepo,
		inspectionRepo: inspectionRepo,
		plotRepo:       plotRepo,
		farmRepo:       farmRepo,
	}
}

func (s *TraceCodeService) GenerateCodes(batchID int64, count int) ([]string, error) {
	// Check batch exists
	batch, err := s.batchRepo.GetByID(batchID)
	if err != nil {
		return nil, fmt.Errorf("batch not found: %w", err)
	}

	// Check if batch is locked
	if batch.Status == model.BatchStatusLocked {
		return nil, fmt.Errorf("batch is locked, cannot generate codes")
	}

	// Check inspection - must have passed
	passed, err := s.inspectionRepo.HasPassedInspection(batchID)
	if err != nil || !passed {
		return nil, fmt.Errorf("batch has not passed inspection")
	}

	// Check safety interval
	ok, msg := s.checkSafetyInterval(batch)
	if !ok {
		return nil, fmt.Errorf("safety interval check failed: %s", msg)
	}

	// Get max seq
	maxSeq, err := s.codeRepo.GetMaxSeqByBatch(batchID)
	if err != nil {
		return nil, err
	}

	// Generate codes in batch of 1000
	var allCodes []string
	batchSize := 1000
	for i := 0; i < count; i += batchSize {
		end := i + batchSize
		if end > count {
			end = count
		}
		size := end - i

		var codes []model.TraceCode
		var codeStrings []string
		for j := 0; j < size; j++ {
			seq := int64(maxSeq + i + j + 1)
			code := tracecode.Generate(seq)
			codes = append(codes, model.TraceCode{
				BatchID: batchID,
				Code:    code,
				Seq:     int(seq),
			})
			codeStrings = append(codeStrings, code)
		}

		if err := s.codeRepo.BatchInsert(codes); err != nil {
			return nil, fmt.Errorf("batch insert codes: %w", err)
		}
		allCodes = append(allCodes, codeStrings...)
	}

	return allCodes, nil
}

// Trace 记录一次扫码并返回扫码页数据。
// 返回体只含核对真伪必需的内容（批次、产地、检测结论、包装单位、扫码防伪信息），
// 不包含操作人、用量、施药次数等个人信息与生产细节。
// region 为服务端解析出的扫码地区，nil 表示未知。
func (s *TraceCodeService) Trace(code string, region *string) (*model.TraceResponse, error) {
	// 原子记录扫码：scan_count 恒加一，首扫时间与地区只在首次写入。
	tc, err := s.codeRepo.RecordScan(code, region)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTraceCodeNotFound
		}
		return nil, fmt.Errorf("record scan: %w", err)
	}
	isFirstScan := tc.ScanCount == 1

	batch, err := s.batchRepo.GetByID(tc.BatchID)
	if err != nil {
		return nil, err
	}

	plot, err := s.plotRepo.GetByID(batch.PlotID)
	if err != nil {
		return nil, err
	}

	farm, err := s.farmRepo.GetByID(plot.FarmID)
	if err != nil {
		return nil, err
	}

	inspection, _ := s.inspectionRepo.GetByBatch(tc.BatchID)

	// 包装单位对应关系：该码是批次内第几件、批次共几件。
	totalUnits, err := s.codeRepo.CountByBatch(tc.BatchID)
	if err != nil {
		return nil, err
	}

	resp := &model.TraceResponse{
		Code:      code,
		FirstScan: isFirstScan,
		ScanCount: tc.ScanCount,
		Batch: &model.TraceBatchInfo{
			CropID:      batch.CropID,
			SowingDate:  batch.SowingDate.Format("2006-01-02"),
			HarvestDate: "",
		},
		Farm: &model.TraceFarmInfo{
			Name:       farm.Name,
			RegionCode: farm.RegionCode,
		},
		PackageUnit: &model.TracePackageUnitInfo{
			Seq:   tc.Seq,
			Total: totalUnits,
		},
	}
	if batch.HarvestDate != nil {
		resp.Batch.HarvestDate = batch.HarvestDate.Format("2006-01-02")
	}

	// 首扫时间与地区用于二次扫码的防伪提示（“此码已于何时何地被扫”）。
	if tc.FirstScannedAt != nil {
		resp.FirstScannedAt = tc.FirstScannedAt.UTC().Format(time.RFC3339)
	}
	if tc.FirstScanRegion != nil {
		resp.FirstScanRegion = *tc.FirstScanRegion
	}

	if inspection != nil {
		resp.Inspection = &model.TraceInspectionInfo{
			Lab:       inspection.Lab,
			SampledAt: inspection.SampledAt.Format("2006-01-02"),
			Result:    inspection.Result,
		}
	}

	return resp, nil
}

func (s *TraceCodeService) checkSafetyInterval(batch *model.CropBatch) (bool, string) {
	if batch.HarvestDate == nil {
		return true, ""
	}

	lastPesticideDate, err := s.batchRepo.GetLastPesticideDate(batch.ID)
	if err != nil || lastPesticideDate == nil {
		return true, ""
	}

	maxInterval, err := s.batchRepo.GetMaxSafeInterval(batch.ID)
	if err != nil || maxInterval == 0 {
		return true, ""
	}

	daysSincePesticide := int(batch.HarvestDate.Sub(*lastPesticideDate).Hours() / 24)
	if daysSincePesticide < maxInterval {
		return false, fmt.Sprintf("距上次施药%d天，不足安全间隔期%d天", daysSincePesticide, maxInterval)
	}
	return true, ""
}

func (s *TraceCodeService) ValidateTraceCode(code string) bool {
	return tracecode.Validate(code)
}
