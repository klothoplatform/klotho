package csharp

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
	"sort"
	"strings"
	"testing"
)

func TestSanitizeConventionalRoute(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path",
			path:     "/hello",
			expected: "/hello",
		},
		{
			name:     "one path param",
			path:     "/hello/{world}",
			expected: "/hello/:world",
		}, {
			name:     "multiple path params",
			path:     "/hello/{other}/{world}",
			expected: "/hello/:other/:world",
		},
		{
			name:     "param with constraint",
			path:     "/hello/{world:int}",
			expected: "/hello/:world",
		},
		{
			name:     "optional param is converted to wildcard",
			path:     "/hello/{world?}",
			expected: "/hello/:rest*",
		},
		{
			name:     "default param is converted to wildcard",
			path:     "/hello/other/{world=default}",
			expected: "/hello/other/:rest*",
		},
		{
			name:     "earlier path params are preserved when converting to wildcard",
			path:     "/hello/{other}/{world=default}",
			expected: "/hello/:other/:rest*",
		},
		{
			name:     "path is converted to longest possible wildcard",
			path:     "/hello/far/away/{other?}/{world=default}",
			expected: "/hello/far/away/:rest*",
		},
		{
			name:     "wildcards are converted to proxy routes 1",
			path:     "/api/{*}",
			expected: "/api/:rest*",
		},
		{
			name:     "wildcards are converted to proxy routes 2",
			path:     "/api/{**slug}",
			expected: "/api/:rest*",
		},
		{
			name:     "wildcards are converted to proxy routes 3",
			path:     "/api/{**slug}/trimmed",
			expected: "/api/:rest*",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizeConventionalPath(tt.path))
		})
	}
}

func Test_sanitizeAttributeBasedPath(t *testing.T) {

	tests := []struct {
		name       string
		path       string
		area       string
		controller string
		action     string
		expected   string
	}{
		{
			name:     "simple path",
			path:     "/hello",
			expected: "/hello",
		},
		{
			name:     "one path param",
			path:     "/hello/{world}",
			expected: "/hello/:world",
		}, {
			name:     "multiple path params",
			path:     "/hello/{other}/{world}",
			expected: "/hello/:other/:world",
		},
		{
			name:     "param with constraint",
			path:     "/hello/{world:int}",
			expected: "/hello/:world",
		},
		{
			name:     "optional param is converted to wildcard",
			path:     "/hello/{world?}",
			expected: "/hello/:rest*",
		},
		{
			name:     "default param is converted to wildcard",
			path:     "/hello/other/{world=default}",
			expected: "/hello/other/:rest*",
		},
		{
			name:     "earlier path params are preserved when converting to wildcard",
			path:     "/hello/{other}/{world=default}",
			expected: "/hello/:other/:rest*",
		},
		{
			name:     "path is converted to longest possible wildcard",
			path:     "/hello/far/away/{other?}/{world=default}",
			expected: "/hello/far/away/:rest*",
		},
		{
			name:       "special tokens are replaced with their in-context values if the specified values match",
			path:       "/api/{area=MyArea}/{controller=MyController}/{action=MyAction}",
			expected:   "/api/MyArea/MyController/MyAction",
			area:       "MyArea",
			controller: "MyController",
			action:     "MyAction",
		},
		{
			name:       "special path params are ignored if their values do not match the current action's values",
			path:       "/api/{area=MyArea}/{controller=OtherController}/{action=OtherAction}",
			expected:   "/api/MyArea/:rest*",
			area:       "MyArea",
			controller: "MyController",
			action:     "MyAction",
		},
		{
			name:       "special tokens are replaced with their in-context values",
			path:       "/api/[area]/[CONTROLLER]/[Action]",
			expected:   "/api/MyArea/MyController/MyAction",
			area:       "MyArea",
			controller: "MyController",
			action:     "MyAction",
		},
		{
			name:     "wildcards are converted to proxy routes 1",
			path:     "/api/{*}",
			expected: "/api/:rest*",
		},
		{
			name:     "wildcards are converted to proxy routes 2",
			path:     "/api/{**slug}",
			expected: "/api/:rest*",
		},
		{
			name:     "wildcards are converted to proxy routes 3",
			path:     "/api/{**slug}/trimmed",
			expected: "/api/:rest*",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizeAttributeBasedPath(tt.path, tt.area, tt.controller, tt.action))
		})
	}
}

func Test_findIApplicationBuilder(t *testing.T) {
	type expectations struct {
		startupClass           string
		appBuilderIdentifier   string
		routeBuilderIdentifier string
	}
	tests := []struct {
		name         string
		program      string
		expectations []expectations
	}{
		{
			name: "Finds Annotated Startup Classes",
			program: `
			using Microsoft.AspNetCore.Builder;
			using Microsoft.AspNetCore.Hosting;
			
			public class MyStartupClass {
				public void Configure(IApplicationBuilder app, IWebHostEnvironment env)
				{
					/**
					* @klotho::expose {
					*  id = "csharp-gateway"
					*  target = "public"
					* }
					*/
					app.UseEndpoints(endpoints =>
					{				
						endpoints.MapGet("/}", () => "Hello!");
					});
				}
			}
			
			public class MyQualifiedStartupClass {
				public void Configure(
					Microsoft.AspNetCore.Builder.IApplicationBuilder qualifiedApp,
					Microsoft.AspNetCore.Hosting.IWebHostEnvironment env)
				{
					/**
					* @klotho::expose {
					*  id = "csharp-gateway2"
					*  target = "public"
					* }
					*/
					qualifiedApp.UseEndpoints(endpoints =>
					{				
						endpoints.MapGet("/}", () => "Hello!");
					});
				}
			}
			
			public class InvalidStartupClassWrongArgType {
				public void Configure(SomeOtherType app, IWebHostEnvironment env)
				{
					/**
					* @klotho::expose {
					*  id = "csharp-gateway3"
					*  target = "public"
					* }
					*/
					app.UseEndpoints(endpoints =>
					{				
						endpoints.MapGet("/}", () => "Hello!");
					});
				}
			}
			
			public class InvalidStartupClassNoConfigureMethod {
				public void OtherMethod(IApplicationBuilder app, IWebHostEnvironment env)
				{
					/**
					* @klotho::expose {
					*  id = "csharp-gateway4"
					*  target = "public"
					* }
					*/
					app.UseEndpoints(endpoints =>
					{				
						endpoints.MapGet("/}", () => "Hello!");
					});
				}
			}
			
			public class InvalidNonAnnotatedStartupClass {
				public void Configure(IApplicationBuilder app, IWebHostEnvironment env)
				{
					app.UseEndpoints(endpoints =>
					{				
						endpoints.MapGet("/}", () => "Hello!");
					});
				}
			}
			
				`,
			expectations: []expectations{
				{
					startupClass:           "MyStartupClass",
					appBuilderIdentifier:   "app",
					routeBuilderIdentifier: "endpoints",
				},
				{
					startupClass:           "MyQualifiedStartupClass",
					appBuilderIdentifier:   "qualifiedApp",
					routeBuilderIdentifier: "endpoints",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			file, err := core.NewSourceFile("program.cs", strings.NewReader(tt.program), Language)
			if !assert.NoError(err) {
				return
			}
			var actual []expectations
			for _, a := range file.Annotations() {
				if a.Capability.Name == "expose" {
					results := findIApplicationBuilder(a)
					for _, r := range results {
						actual = append(actual, expectations{
							startupClass:           r.StartupClassDeclaration.ChildByFieldName("name").Content(),
							appBuilderIdentifier:   r.AppBuilderIdentifier.Content(),
							routeBuilderIdentifier: r.EndpointRouteBuilderIdentifier.Content(),
						})
					}
				}
			}

			sort.Slice(tt.expectations, func(i, j int) bool {
				return tt.expectations[i].startupClass < tt.expectations[j].startupClass
			})
			sort.Slice(actual, func(i, j int) bool {
				return actual[i].startupClass < actual[j].startupClass
			})

			assert.Equal(len(tt.expectations), len(actual), "Incorrect number of results")
			assert.Equal(tt.expectations, actual)
		})
	}
}
