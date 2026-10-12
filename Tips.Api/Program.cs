using Asp.Versioning;
using Asp.Versioning.Builder;
using Tips.Api;
using Tips.Api.Endpoints;
using Tips.Api.Settings;

WebApplicationBuilder builder = WebApplication.CreateBuilder(args);

builder
    .AddApplicationServices()
    .AddBackgroundJobs()
    .AddCorsPolicy()
    .AddRateLimiting()
    .AddErrorHandling();

WebApplication app = builder.Build();

app.UseExceptionHandler();
app.UseStatusCodePages();

if (!app.Environment.IsDevelopment())
{
    app.UseHttpsRedirection();
}

app.UseCors(CorsOptions.PolicyName);
app.UseRateLimiter();
app.UseOutputCache();

ApiVersionSet apiVersionSet = app.NewApiVersionSet()
    .HasApiVersion(new ApiVersion(1))
    .ReportApiVersions()
    .Build();

RouteGroupBuilder versionedGroup = app
    .MapGroup("api/v{version:apiVersion}")
    .WithApiVersionSet(apiVersionSet);

versionedGroup
    .MapTipsEndpoints()
    .MapLeaderboardEndpoints();

app.MapHealthChecks("/health");

if (app.Environment.IsDevelopment())
{
    app.MapOpenApi().WithDocumentPerVersion();
    app.UseSwaggerUI(options => options.SwaggerEndpoint("/openapi/v1.json", "Tips API v1"));
}

await app.RunAsync();

/// <summary>Exposed so integration tests can use WebApplicationFactory.</summary>
public partial class Program;
