package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func GetMultiEnvVar(envVars ...string) (string, error) {
	for _, value := range envVars {
		if v := os.Getenv(value); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("unable to retrieve any env vars from list: %v", envVars)
}

func IsHTTPSuccess(response *http.Response) bool {
	if response == nil {
		return false
	}
	return response.StatusCode >= 200 && response.StatusCode < 300
}

func GetRespErrorDetail(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

func NewNotFoundErrorDiagnostic(clusterName, orgName string) diag.Diagnostic {
	return diag.NewErrorDiagnostic("cluster not found", fmt.Sprintf("cluster %s in org %s not found", clusterName, orgName))
}

// ExtractAttrValue recursively extracts a framework attr.Value into pure Go types
func ExtractAttrValue(ctx context.Context, v attr.Value) interface{} {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}

	switch val := v.(type) {
	case basetypes.StringValue:
		return val.ValueString()
	case basetypes.BoolValue:
		return val.ValueBool()
	case basetypes.Int64Value:
		return val.ValueInt64()
	case basetypes.Float64Value:
		return val.ValueFloat64()
	case basetypes.NumberValue:
		if f, _ := val.ValueBigFloat().Float64(); true {
			return f
		}
		return nil
	case basetypes.ListValue:
		var slice []interface{}
		for _, item := range val.Elements() {
			slice = append(slice, ExtractAttrValue(ctx, item))
		}
		return slice
	case basetypes.MapValue:
		m := make(map[string]interface{})
		for k, item := range val.Elements() {
			m[k] = ExtractAttrValue(ctx, item)
		}
		return m
	case basetypes.ObjectValue:
		m := make(map[string]interface{})
		for k, item := range val.Attributes() {
			m[k] = ExtractAttrValue(ctx, item)
		}
		return m
	case basetypes.DynamicValue:
		return ExtractAttrValue(ctx, val.UnderlyingValue())
	default:
		// Attempt fallback via JSON string parsing if string cast exists, else nil
		return nil
	}
}
