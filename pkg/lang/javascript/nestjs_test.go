package javascript

import (
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
)

/*
This test should validate multiple things

The queries; nestJsRoute, nestJsController, nestJsModule (based on test case)
The validation for each resource; controller, route, module (based on test case)
query.FindReference will be tested by default.
*/
func Test_nestHandler_QueryResources(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		numResources map[string]int
	}{
		{
			name: "simple Module",
			source: `const common_1 = require("@nestjs/common");
			AppModule = __decorate([
				(0, common_1.Module)({
					imports: [],
					controllers: [app_controller_1.UsersController, app_controller_1.OrgController],
					providers: [app_service_1.AppService],
				})
			], AppModule);`,
			numResources: map[string]int{
				"modules":     3, // We actually match 3 here because we look for each match of pairs that are member expressions. Then in the code we check the key of that pair to ensure its controllers
				"controllers": 0,
				"routes":      0,
			},
		},
		{
			name: "simple Module, fails import validation",
			source: `const common_1 = require("@nestjs/common");
			AppModule = __decorate([
				(0, not_common_1.Module)({
					imports: [],
					controllers: [app_controller_1.UsersController, app_controller_1.OrgController],
					providers: [app_service_1.AppService],
				})
			], AppModule);`,
			numResources: map[string]int{
				"modules":     0,
				"factories":   0,
				"controllers": 0,
				"routes":      0,
			},
		},
		{
			name: "simple Module, fails method validation",
			source: `const common_1 = require("@nestjs/common");
			AppModule = __decorate([
				(0, common_1.NOTModule)({
					imports: [],
					controllers: [app_controller_1.UsersController, app_controller_1.OrgController],
					providers: [app_service_1.AppService],
				})
			], AppModule);`,
			numResources: map[string]int{
				"modules":     0,
				"controllers": 0,
				"routes":      0,
			},
		},
		{
			name: "simple Controller",
			source: `const common_1 = require("@nestjs/common");
			UsersController = __decorate([
				(0, common_1.Controller)('users'),
				__metadata("design:paramtypes", [app_service_1.AppService])
			], UsersController);`,
			numResources: map[string]int{
				"modules":     0,
				"controllers": 1,
				"routes":      0,
			},
		},
		{
			name: "simple Controller, fails import validation",
			source: `const common_1 = require("@nestjs/uncommon");
			UsersController = __decorate([
				(0, common_1.Controller)('users'),
				__metadata("design:paramtypes", [app_service_1.AppService])
			], UsersController);`,
			numResources: map[string]int{
				"modules":     0,
				"controllers": 0,
				"routes":      0,
			},
		},
		{
			name: "simple Controller, fails method validation",
			source: `const common_1 = require("@nestjs/common");
			UsersController = __decorate([
				(0, common_1.NotController)('users'),
				__metadata("design:paramtypes", [app_service_1.AppService])
			], UsersController);`,
			numResources: map[string]int{
				"modules":     0,
				"controllers": 0,
				"routes":      0,
			},
		},
		{
			name: "simple Route",
			source: `const common_1 = require("@nestjs/common");
			__decorate([
				(0, common_1.Get)(':id'),
				__metadata("design:type", Function),
				__metadata("design:paramtypes", []),
				__metadata("design:returntype", String)
			], OrgController.prototype, "getOrg", null);`,
			numResources: map[string]int{
				"modules":     0,
				"controllers": 0,
				"routes":      1,
			},
		},
		{
			name: "simple Route, fails import validation",
			source: `const common_1 = require("@nestjs/common");
			__decorate([
				(0, nothing.Get)(':id'),
				__metadata("design:type", Function),
				__metadata("design:paramtypes", []),
				__metadata("design:returntype", String)
			], OrgController.prototype, "getOrg", null);`,
			numResources: map[string]int{
				"modules":     0,
				"controllers": 0,
				"routes":      0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			f, err := NewFile("", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}
			var testNestHandler = &NestJsHandler{
				log: zap.L(),
			}
			testNestHandler.queryResources(f)

			assert.Len(testNestHandler.output.modules, tt.numResources["modules"])
			assert.Len(testNestHandler.output.controllers, tt.numResources["controllers"])
			assert.Len(testNestHandler.output.routes, tt.numResources["routes"])
		})
	}
}

func Test_nestHandler_FindFactory(t *testing.T) {
	tests := []struct {
		name   string
		source string
		expect map[string]string
	}{
		{
			name: "simple Factory",
			source: `
			const core_1 = require("@nestjs/core");
			const app_module_1 = require("./src/app.module");
			const app = await core_1.NestFactory.create(app_module_1.AppModule, new platform_express_1.ExpressAdapter(expressApp));
			`,
			expect: map[string]string{
				"varName":          "app",
				"moduleImportName": "AppModule",
				"moduleImportPath": "./src/app.module",
			},
		},
		{
			name: "Factory Invalid import",
			source: `
			const core_1 = require("@nestjs/notCore");
			const app_module_1 = require("./src/app.module");
			const app = await core_1.NestFactory.create(app_module_1.AppModule, new platform_express_1.ExpressAdapter(expressApp));
			`,
			expect: make(map[string]string),
		},
		{
			name: "Factory Invalid call",
			source: `
			const core_1 = require("@nestjs/core");
			const app_module_1 = require("./src/app.module");
			const app = await core_1.NestFactory.random(app_module_1.AppModule, new platform_express_1.ExpressAdapter(expressApp));
			`,
			expect: make(map[string]string),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			f, err := NewFile("", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}
			var testNestHandler = &NestJsHandler{
				log: zap.L(),
			}
			factory := testNestHandler.findNestFactory(f)
			assert.Equal(tt.expect["moduleImportName"], factory.moduleImportName)
			assert.Equal(tt.expect["varName"], factory.varName)
			assert.Equal(tt.expect["moduleImportPath"], factory.moduleImportPath)
		})
	}
}
