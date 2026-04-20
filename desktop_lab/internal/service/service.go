package service

import (
	"desktop_lab/internal/repository"

	"go.uber.org/zap"
)

type Services struct {
	Materials  *MaterialService
	Standards  *StandardService
	Protocols  *ProtocolService
	Groups     *ExperimentGroupService
	Samples    *SampleService
	Reports    *ReportService
	Dimensions *DimensionService
}

func NewServices(
	matRepo repository.MaterialRepo,
	stdRepo repository.StandardRepo,
	protRepo repository.ProtocolRepo,
	sampRepo repository.SampleRepo,
	groupRepo repository.ExperimentGroupRepo,
	dimRepo repository.DimensionRepo,
	fontDir string,
	templatesDir string,
	log *zap.Logger,
) *Services {
	material := NewMaterialService(matRepo, log)
	standards := NewStandardService(stdRepo, log)
	sample := NewSampleService(sampRepo, log)
	group := NewExperimentGroupService(groupRepo, log)
	protocol := NewProtocolService(protRepo, sampRepo, stdRepo, groupRepo, matRepo, log)
	report := NewReportService(protocol, material, fontDir, templatesDir, log)
	dimension := NewDimensionService(dimRepo, log)
	return &Services{
		Materials:  material,
		Standards:  standards,
		Protocols:  protocol,
		Groups:     group,
		Samples:    sample,
		Reports:    report,
		Dimensions: dimension,
	}
}
