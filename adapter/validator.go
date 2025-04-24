package adapter

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateSpecFromSchema 使用 OpenAPI 模式 GVK 定义自动验证规范图
func ValidateSpecFromSchema(spec map[string]interface{}, group, version, kind string) error {
	gvkKey := BuildGVKKey(group, version, kind)
	fields, err := ExtractSpecFieldsFromOpenAPI(gvkKey)
	if err != nil {
		return fmt.Errorf("failed to extract spec fields: %w", err)
	}
	return ValidateSpecFields(spec, fields)
}

// ValidateSpecFields 验证给定的规范字段是否在允许的字段列表中
func ValidateSpecFields(spec map[string]interface{}, validFields []string) error {
	var errs field.ErrorList
	validSet := make(map[string]struct{}, len(validFields))
	for _, f := range validFields {
		validSet[f] = struct{}{}
	}
	for k := range spec {
		if _, ok := validSet[k]; !ok {
			errs = append(errs, field.Invalid(field.NewPath("spec").Key(k), k, "not allowed by OpenAPI schema"))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errs.ToAggregate()
}
