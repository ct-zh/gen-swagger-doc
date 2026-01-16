package generator

import (
	"strings"
	"testing"

	"gen-swagger-doc/pkg/driver"
)

func TestGenerateSwaggerDocs(t *testing.T) {
	tests := []struct {
		name      string
		route     driver.RouteInfo
		params    []driver.ParamInfo
		responses []driver.ResponseInfo
		want      []string
	}{
		{
			name: "Basic Route",
			route: driver.RouteInfo{
				Method: "GET",
				Path:   "/users",
			},
			want: []string{
				"@Summary GET /users",
				"@Accept json",
				"@Produce json",
				"@Success 200 {object} map[string]interface{}",
				"@Router /users [get]",
			},
		},
		{
			name: "With Params and Responses",
			route: driver.RouteInfo{
				Method: "POST",
				Path:   "/users",
			},
			params: []driver.ParamInfo{
				{Name: "id", In: "path", Type: "int", Required: true, Desc: "User ID"},
				{Name: "user", In: "body", Type: "object", Required: true, Desc: "User Info", Schema: "UserRequest"},
			},
			responses: []driver.ResponseInfo{
				{Code: 200, Schema: "UserResponse", IsArray: false, Desc: "Success"},
				{Code: 400, Schema: "ErrorResponse", IsArray: false, Desc: "Bad Request"},
			},
			want: []string{
				"@Param id path int true \"User ID\"",
				"@Param user body object true \"User Info\" UserRequest",
				"@Success 200 {object} UserResponse \"Success\"",
				"@Success 400 {object} ErrorResponse \"Bad Request\"",
				"@Router /users [post]",
			},
		},
		{
			name: "With Empty Description Params",
			route: driver.RouteInfo{
				Method: "GET",
				Path:   "/test",
			},
			params: []driver.ParamInfo{
				{Name: "uid", In: "query", Type: "integer", Required: true, Desc: ""},
			},
			want: []string{
				"@Param uid query integer true \"uid\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSwaggerDocs(tt.route, tt.params, tt.responses)
			
			// 验证包含所有预期的行
			for _, wantLine := range tt.want {
				found := false
				for _, gotLine := range got {
					if strings.TrimSpace(gotLine) == wantLine {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GenerateSwaggerDocs() missing line = %v\nGot:\n%v", wantLine, strings.Join(got, "\n"))
				}
			}
		})
	}
}
