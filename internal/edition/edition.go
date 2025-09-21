// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
package edition

type Edition string

const EditionCE Edition = "ce"

// 可选：构建时 ldflags 注入 Version；若无则保留默认
var BuildEdition = string(EditionCE)

type FeatureFlag string

const (
	FeatureTemporalValidation FeatureFlag = "temporal-validation"
	FeatureSemanticValidation FeatureFlag = "semantic-validation"
	FeatureDAGValidation      FeatureFlag = "dag-validation"
	FeatureHTMLReport         FeatureFlag = "html-report"
	FeatureJSONReport         FeatureFlag = "json-report"
	FeatureJUnitReport        FeatureFlag = "junit-report"
	FeatureBaselineBasic      FeatureFlag = "baseline-basic"
	FeatureDiscoverBasic      FeatureFlag = "discover-basic"
)

func Current() Edition { return EditionCE }

func (ed Edition) Supports(f FeatureFlag) bool {
	switch f {
	case FeatureTemporalValidation,
		FeatureSemanticValidation,
		FeatureDAGValidation,
		FeatureHTMLReport,
		FeatureJSONReport,
		FeatureJUnitReport,
		FeatureBaselineBasic,
		FeatureDiscoverBasic:
		return true
	default:
		return false
	}
}