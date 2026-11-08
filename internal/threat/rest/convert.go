package rest

import (
	"github.com/andrey-kalistratov/threat-info-service/internal/rest/generated"
	"github.com/andrey-kalistratov/threat-info-service/internal/threat"
)

type threatRequestData interface {
	GetName() string
	GetDescription() generated.OptString
	GetConfidence() generated.OptInt32
	GetMalwareTypes() []generated.MalwareType
}

func threatFromCreateRequest(body *generated.Threat) (*threat.Threat, error) {
	return threatFromRequest(body.MD5, body)
}

func threatFromUpdateRequest(body *generated.ThreatData, params generated.UpdateThreatParams) (*threat.Threat, error) {
	return threatFromRequest(params.MD5, body)
}

func threatFromRequest(md5 generated.MD5, body threatRequestData) (*threat.Threat, error) {
	parsedMD5, err := md5FromRequest(md5)
	if err != nil {
		return nil, err
	}

	value := &threat.Threat{
		MD5:          parsedMD5,
		Name:         body.GetName(),
		Description:  body.GetDescription().Or(""),
		MalwareTypes: malwareTypesFromRequest(body.GetMalwareTypes()),
	}

	confidence := body.GetConfidence()
	if score, ok := confidence.Get(); ok {
		value.Confidence = new(int(score))
	}

	return value, nil
}

func md5FromRequest(value generated.MD5) (threat.MD5, error) {
	return threat.ParseMD5(string(value))
}

func malwareTypesFromRequest(categories []generated.MalwareType) []threat.MalwareType {
	result := make([]threat.MalwareType, len(categories))
	for i, category := range categories {
		result[i] = threat.MalwareType(category)
	}
	return result
}

func threatToResponse(value *threat.Threat) *generated.Threat {
	result := &generated.Threat{
		MD5:          generated.MD5(value.MD5),
		Name:         value.Name,
		Description:  generated.NewOptString(value.Description),
		MalwareTypes: make([]generated.MalwareType, len(value.MalwareTypes)),
	}

	if value.Confidence != nil {
		//nolint:gosec // Доменный инвариант threat.Threat: confidence в диапазоне 0–100.
		score := int32(*value.Confidence)
		result.Confidence = generated.NewOptInt32(score)
	}

	for i, category := range value.MalwareTypes {
		result.MalwareTypes[i] = generated.MalwareType(category)
	}

	return result
}
