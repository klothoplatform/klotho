package javascript

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_expose_findApp(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		expect    string
		expectErr bool
	}{
		{
			name: "simple listen",
			source: `const app = express();
// @klotho::expose
app.listen(3000);`,
			expect: "app",
		},
		{
			name: "setup function",
			source: `
async function setup() {
	const app = express();
	await configure(app);
	return app;
}
const appPromise = setup();

appPromise.then(app => {
	// @klotho::expose
	app.listen(3000);
});`,
			expect: "appPromise",
		},
		{
			name: "listen in setup function",
			source: `
async function setup() {
	const app = express();
	await configure(app);

	// @klotho::expose
	app.listen(3000);

	return app;
}
const appPromise = setup();
`,
			expect: "appPromise",
		},
		{
			name: "listen in setup function not assigned",
			source: `
async function setup() {
	const app = express();
	await configure(app);

	// @klotho::expose
	app.listen(3000);
}
`,
			expectErr: true,
		},
		{
			name: "setup function",
			source: `
			async function setup() {
				const expressApp = express();
				const app = await core_1.NestFactory.create(app_module_1.AppModule, new platform_express_1.ExpressAdapter(expressApp));
				await app.init();
				return expressApp;
			}
			const appPromise = setup();

			appPromise.then(app => {
				// @klotho::expose
				app.listen(3000);
			});`,
			expect: "appPromise",
		},
		{
			name: "listen in setup function",
			source: `
			async function bootstrap() {
				const expressApp = express();
				const app = await core_1.NestFactory.create(app_module_1.AppModule, new platform_express_1.ExpressAdapter(expressApp));
				await app.init();
				/**
				 * @klotho::expose {
				 *  id = "orgApp"
				 *  target = "public"
				 * }
				 */
				await app.listen(3000);
				return expressApp;
			}
			const app = bootstrap();
`,
			expect: "app",
		},
		{
			name: "listen in setup function not assigned",
			source: `
			async function setup() {
				const app = express();
				await configure(app);

				// @klotho::expose
				app.listen(3000);
			}
			`,
			expectErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			f, err := NewFile("", strings.NewReader(tt.source))
			if !assert.NoError(err) {
				return
			}
			var annot *core.Annotation
			for _, v := range f.Annotations() {
				annot = v
				break
			}
			listen := findListener(annot, f.Program())
			if !assert.NotNil(listen.Expression, "error in test source listen function") {
				return
			}

			got, err := findApp(f.Program(), listen)
			if tt.expectErr {
				assert.Error(err)
				return
			}
			assert.Equal(tt.expect, got)
		})
	}
}
