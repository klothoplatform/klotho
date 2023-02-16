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

func TestExpose_Transform(t *testing.T) {
	type gateway struct {
		Name   string
		Routes []routeMethodPath
	}
	type srcFile struct {
		Path    string
		Content string
	}

	parseDep := func(dep string) core.Dependency {
		parts := strings.Split(dep, ":")
		return core.Dependency{
			Source: core.ResourceKey{Kind: "gateway", Name: parts[0]},
			Target: core.ResourceKey{Kind: "exec_unit", Name: parts[1]},
		}
	}

	tests := []struct {
		name             string
		units            map[string][]srcFile
		expectedGateways []gateway
		expectedDeps     []string
	}{
		{
			name: "Routes added using Map<VERB>() are detected",
			units: map[string][]srcFile{
				"main": {
					{
						Path: "Startup.cs",
						Content: `
						using Microsoft.AspNetCore.Builder;
						using Microsoft.AspNetCore.Hosting;
						using Microsoft.AspNetCore.Http;
						using Microsoft.AspNetCore.Routing;

						namespace WebAPILambda
						{
							public class Startup
							{
								public void Configure(IApplicationBuilder app, IWebHostEnvironment env)
								{
									/**
									 * @klotho::expose {
									 *  id = "my-gateway"
									 *  target = "public"
									 * }
									 */
									app.UseEndpoints(endpoints =>
									{
										endpoints.MapGet("/path", () => "ok");
										endpoints.MapPut("/path", () => "ok");
										endpoints.MapPost("/path", () =>  "ok");
										endpoints.MapDelete("/other-path", () => "ok");
									});
								}
							}
						}
						`,
					},
				},
			},
			expectedGateways: []gateway{
				{
					Name: "my-gateway",
					Routes: []routeMethodPath{
						{Verb: core.VerbGet, Path: "/path"},
						{Verb: core.VerbPost, Path: "/path"},
						{Verb: core.VerbPut, Path: "/path"},
						{Verb: core.VerbDelete, Path: "/other-path"},
					},
				},
			},
			expectedDeps: []string{
				"my-gateway:main",
			},
		},
		{
			name: "Controller routes are added if AddControllers() and MapControllers() are invoked ",
			units: map[string][]srcFile{
				"unit1-MapControllers": {
					{
						Path: "Startup.cs",
						Content: `
						using Microsoft.AspNetCore.Builder;
						using Microsoft.AspNetCore.Hosting;
						using Microsoft.AspNetCore.Http;
						using Microsoft.Extensions.Configuration;
						using Microsoft.Extensions.DependencyInjection;
						using Microsoft.Extensions.Hosting;

						namespace WebAPILambda
						{
							public class Startup
							{							
								public void ConfigureServices(IServiceCollection services)
								{
									services.AddControllers();
								}
						
								public void Configure(IApplicationBuilder app, IWebHostEnvironment env)
								{
									/**
									 * @klotho::expose {
									 *  id = "gateway1"
									 *  target = "public"
									 * }
									 */
									app.UseEndpoints(endpoints =>
									{
										endpoints.MapGet("/local-route", () => "ok");
										endpoints.MapControllers();
									});
								}
							}
						}
						`,
					},
					{
						Path: "controller1.cs",
						Content: `
						using System;
						using Microsoft.AspNetCore.Mvc;
						
						namespace WebAPILambda.Controllers
						{
													
							[Route("api/[controller]")]
							public class Controller1Controller
							{
								[HttpGet]
								public string Get()
								{
									return "ok";
								}
							}
						}
						`,
					},
				},
				"unit2-no-MapControllers": {
					{
						Path: "Startup.cs",
						Content: `
						using Microsoft.AspNetCore.Builder;
						using Microsoft.AspNetCore.Hosting;
						using Microsoft.AspNetCore.Http;
						using Microsoft.Extensions.Configuration;
						using Microsoft.Extensions.DependencyInjection;
						using Microsoft.Extensions.Hosting;

						namespace WebAPILambda
						{
							public class Startup
							{							
								public void ConfigureServices(IServiceCollection services)
								{
									services.AddControllers();
								}
						
								public void Configure(IApplicationBuilder app, IWebHostEnvironment env)
								{
									/**
									 * @klotho::expose {
									 *  id = "gateway2"
									 *  target = "public"
									 * }
									 */
									app.UseEndpoints(endpoints =>
									{
										endpoints.MapGet("/local-route", () => "ok");
									});
								}
							}
						}
						`,
					},
					{
						Path: "controller1.cs",
						Content: `
						using System;
						using Microsoft.AspNetCore.Mvc;
						
						namespace WebAPILambda.Controllers
						{
													
							[Route("api/[controller]")]
							public class Controller1Controller
							{
								[HttpGet]
								public string Get()
								{
									return "ok";
								}
							}
						}
						`,
					},
				},
				"unit3-no-AddControllers": {
					{
						Path: "Startup.cs",
						Content: `
						using Microsoft.AspNetCore.Builder;
						using Microsoft.AspNetCore.Hosting;
						using Microsoft.AspNetCore.Http;
						using Microsoft.Extensions.Configuration;
						using Microsoft.Extensions.DependencyInjection;
						using Microsoft.Extensions.Hosting;

						namespace WebAPILambda
						{
							public class Startup
							{
								public void Configure(IApplicationBuilder app, IWebHostEnvironment env)
								{
									/**
									 * @klotho::expose {
									 *  id = "gateway3"
									 *  target = "public"
									 * }
									 */
									app.UseEndpoints(endpoints =>
									{
										endpoints.MapControllers();
									});
								}
							}
						}
						`,
					},
					{
						Path: "controller1.cs",
						Content: `
						using System;
						using Microsoft.AspNetCore.Mvc;
						
						namespace WebAPILambda.Controllers
						{
													
							[Route("api/[controller]")]
							public class Controller1Controller
							{
								[HttpGet]
								public string Get()
								{
									return "ok";
								}
							}
						}
						`,
					},
				},
			},
			expectedGateways: []gateway{
				{
					Name: "gateway1",
					Routes: []routeMethodPath{
						{Verb: core.VerbGet, Path: "/local-route"},
						{Verb: core.VerbGet, Path: "/api/controller1"},
					},
				},
				{
					Name: "gateway2",
					Routes: []routeMethodPath{
						{Verb: core.VerbGet, Path: "/local-route"},
					},
				},
				{
					Name: "gateway3",
					Routes: []routeMethodPath{
						{Verb: core.VerbAny, Path: "/"},
						{Verb: core.VerbAny, Path: "/:proxy*"},
					},
				},
			},
			expectedDeps: []string{
				"gateway1:unit1-MapControllers",
				"gateway2:unit2-no-MapControllers",
				"gateway3:unit3-no-AddControllers",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			result := &core.CompilationResult{}
			for uName, files := range tt.units {
				unit := &core.ExecutionUnit{
					Name:       uName,
					Executable: core.NewExecutable(),
				}
				for _, f := range files {
					sf, err := core.NewSourceFile(f.Path, strings.NewReader(f.Content), Language)
					if !assert.NoError(err) {
						return
					}
					unit.AddSourceFile(sf)
				}
				result.Add(unit)
			}
			deps := &core.Dependencies{}
			expose := Expose{}
			expose.Transform(result, deps)

			gateways := core.GetResourcesOfType[*core.Gateway](result)
			assert.Equal(len(tt.expectedGateways), len(gateways))

			sort.Slice(gateways, func(i, j int) bool {
				return gateways[i].Name < gateways[j].Name
			})
			sort.Slice(tt.expectedGateways, func(i, j int) bool {
				return tt.expectedGateways[i].Name < tt.expectedGateways[j].Name
			})

			for _, gw := range gateways {
				sort.Slice(gw.Routes, func(i, j int) bool {
					if gw.Routes[i].Path == gw.Routes[j].Path {
						return gw.Routes[i].Verb < gw.Routes[j].Verb
					} else {
						return gw.Routes[i].Path < gw.Routes[j].Path
					}
				})
			}

			for _, gw := range tt.expectedGateways {
				sort.Slice(gw.Routes, func(i, j int) bool {
					if gw.Routes[i].Path == gw.Routes[j].Path {
						return gw.Routes[i].Verb < gw.Routes[j].Verb
					} else {
						return gw.Routes[i].Path < gw.Routes[j].Path
					}
				})
			}

			for i, expectedGw := range tt.expectedGateways {
				if i >= len(gateways) {
					break
				}
				assert.Equal(len(expectedGw.Routes), len(gateways[i].Routes))
				for j, eRoute := range expectedGw.Routes {
					if j >= len(gateways[i].Routes) {
						break
					}
					aRoute := gateways[i].Routes[j]
					assert.Equal(eRoute.Verb, aRoute.Verb)
					assert.Equal(aRoute.Path, aRoute.Path)
				}
			}
			depsArr := deps.ToArray()

			assert.Equal(len(tt.expectedDeps), len(depsArr))
			var eDeps []core.Dependency
			for _, dep := range tt.expectedDeps {
				eDeps = append(eDeps, parseDep(dep))
			}
			sort.Slice(eDeps, func(i, j int) bool {
				if eDeps[i].Source.Name == eDeps[j].Source.Name {
					return eDeps[i].Target.Name < eDeps[j].Target.Name
				} else {
					return eDeps[i].Source.Name < eDeps[j].Source.Name
				}
			})
			sort.Slice(depsArr, func(i, j int) bool {
				if depsArr[i].Source.Name == depsArr[j].Source.Name {
					return depsArr[i].Target.Name < depsArr[j].Target.Name
				} else {
					return depsArr[i].Source.Name < depsArr[j].Source.Name
				}
			})

			for i, eDep := range eDeps {
				if i >= len(depsArr) {
					break
				}
				aDep := depsArr[i]
				assert.Equal(eDep, aDep)
			}
		})
	}
}
