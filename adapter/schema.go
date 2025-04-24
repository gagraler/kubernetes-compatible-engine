package adapter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

var (
	openAPISchemaCache map[string]interface{}
	openAPISchemaOnce  sync.Once
)

// FetchOpenAPISchema 从 Kubernetes API 服务器获取 OpenAPI 模式
func FetchOpenAPISchema(apiServer, token string) (map[string]interface{}, error) {
	var err error
	openAPISchemaOnce.Do(func() {
		url := fmt.Sprintf("%s/openapi/v2", strings.TrimSuffix(apiServer, "/"))
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")
		client := &http.Client{}

		resp, httpErr := client.Do(req)
		if httpErr != nil {
			err = httpErr
			return
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				return
			}
		}(resp.Body)

		var result map[string]interface{}
		if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
			err = decodeErr
			return
		}
		openAPISchemaCache = result
	})
	return openAPISchemaCache, err
}

// ExtractSpecFieldsFromOpenAPI 从 OpenAPI 模式中提取 .spec 字段
func ExtractSpecFieldsFromOpenAPI(gvk string) ([]string, error) {
	defs, ok := openAPISchemaCache["definitions"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("OpenAPI: no definitions found")
	}
	schema, ok := defs[gvk].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("GVK %s not found in schema", gvk)
	}
	specProp, ok := schema["properties"].(map[string]interface{})["spec"]
	if !ok {
		return nil, fmt.Errorf(".spec not found in schema for %s", gvk)
	}
	specFields, ok := specProp.(map[string]interface{})["properties"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf(".spec has no sub-properties in %s", gvk)
	}

	var fields []string
	for field := range specFields {
		fields = append(fields, field)
	}
	return fields, nil
}

// BuildGVKKey 构建 GVK 键
func BuildGVKKey(group, version, kind string) string {
	if group == "" || group == "core" {
		group = "core"
	}
	return fmt.Sprintf("io.k8s.api.%s.%s.%s", group, version, kind)
}
