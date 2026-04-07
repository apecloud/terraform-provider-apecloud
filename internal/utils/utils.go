package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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
