using Microsoft.AspNetCore.Hosting;
using Microsoft.Extensions.Hosting;
using Amazon.Lambda.Core;

namespace KlothoRuntime
{
    public class APIGatewayLambdaDispatcher : {{if .Expose.APIGatewayProxyFunction}}{{.Expose.APIGatewayProxyFunction}}{{else}}{{.Expose.FunctionType}}{{end}}
    {
        protected override void Init(IWebHostBuilder builder)
        {
            LambdaLogger.Log("Initializing execution unit: {{.ExecUnitName}}");
//TMPL {{if .Expose.APIGatewayProxyFunction }}
            base.Init(builder);
//TMPL {{else if .Expose.StartupClass }}
            builder.UseStartup<{{.Expose.StartupClass}}>();
//TMPL {{else}}
            LambdaLogger.Log("{{.ExecUnitName}} is No-Op: No handler detected");
//TMPL {{end}}
        }

        protected override void Init(IHostBuilder builder)
        {
//TMPL {{if .Expose.APIGatewayProxyFunction }}
            base.Init(builder);
//TMPL {{end}}
        }
    }
}
