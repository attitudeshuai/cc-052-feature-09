package service

import (
	"cc-052/internal/model"
	"cc-052/internal/repository"
	"cc-052/pkg/tracecode"
	"database/sql"
	"errors"
	"fmt"
)

type TraceCodeService struct {
	codeRepo       *repository.TraceCodeRepo
	batchRepo      *repository.BatchRepo
	inspectionRepo *repository.InspectionRepo
	plotRepo       *repository.PlotRepo
	farmRepo       *repository.FarmRepo
}

// ErrInvalidPackageUnit 请求发码时传入了字典中不存在的包装单位
var ErrInvalidPackageUnit = errors.New("invalid package unit")

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

// GenerateCodes 为一个批次生成溯源码。每个批次发码时必须指定包装单位，
// 包装单位必须在 package_unit 字典内，保证「一条码↔一包装单位」可查。
func (s *TraceCodeService) GenerateCodes(batchID int64, count int, packageUnitCode string) ([]string, error) {
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

	// Validate package unit against dictionary
	unit, err := s.codeRepo.GetPackageUnit(packageUnitCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidPackageUnit, packageUnitCode)
		}
		return nil, err
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
				BatchID:         batchID,
				Code:            code,
				Seq:             int(seq),
				PackageUnitCode: unit.Code,
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

// Trace 是公开扫码查询。
//
// 出于个人信息保护，返回体只保留核对真伪必需的内容（码本身、包装单位、
// 批次/地块/合作社、检测结论、首次扫码留痕），不再返回农事活动明细——
// 操作人姓名、投入品用量、施药次数等一律不出现在扫码页。
//
// region 是扫码端显式上报的行政区划码；为 nil 表示未知，严禁用请求来源 IP 冒充。
func (s *TraceCodeService) Trace(code string, region *string) (*model.TraceResponse, error) {
	tc, unit, err := s.codeRepo.GetByCode(code)
	if err != nil {
		return nil, fmt.Errorf("code not found: %w", err)
	}

	// 每次扫码都留痕：原子区分首次/再次，并拿到累计扫码次数
	isFirstScan, scanCount, firstScannedAt, firstRegion, err := s.codeRepo.RecordScan(tc.ID, region)
	if err != nil {
		return nil, fmt.Errorf("record scan: %w", err)
	}

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

	resp := &model.TraceResponse{
		Code: code,
		PackageUnit: &model.TracePackageInfo{
			Code: unit.Code,
			Name: unit.Name,
		},
		Batch: &model.TraceBatchInfo{
			CropID:      batch.CropID,
			SowingDate:  batch.SowingDate.Format("2006-01-02"),
			HarvestDate: "",
		},
		Farm: &model.TraceFarmInfo{
			Name:       farm.Name,
			RegionCode: farm.RegionCode,
			PlotName:   plot.Name,
		},
		FirstScan:       isFirstScan,
		ScanCount:       scanCount,
		FirstScannedAt:  firstScannedAt.Format("2006-01-02 15:04:05"),
		FirstScanRegion: "",
	}
	if batch.HarvestDate != nil {
		resp.Batch.HarvestDate = batch.HarvestDate.Format("2006-01-02")
	}
	if firstRegion != nil {
		resp.FirstScanRegion = *firstRegion
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
