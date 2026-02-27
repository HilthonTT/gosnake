using Tips.Api;
using Tips.Api.Endpoints;
using Tips.Api.Settings;
using Asp.Versioning;
using Asp.Versioning.Builder;

WebApplicationBuilder builder = WebApplication.CreateBuilder(args);

builder
    .AddApplicationServices()
    .AddBackgroundJobs()
    .AddCorsPolicy()
    .AddRateLimiting()
    .AddErrorHandling();

WebApplication app = builder.Build();

ApiVersionSet apiVersionSet = app.NewApiVersionSet()
    .HasApiVersion(new ApiVersion(1))
    .ReportApiVersions()
    .Build();

RouteGroupBuilder versionedGroup = app
    .MapGroup("api/v{version:apiVersion}")
    .WithApiVersionSet(apiVersionSet);

app.MapTipsEndpoints(versionedGroup);

app.MapLeaderboardEndpoints(versionedGroup);

if (app.Environment.IsDevelopment())
{
    app.MapOpenApi();
    app.UseSwagger();
    app.UseSwaggerUI();
}

app.UseCookiePolicy();

app.UseCors(CorsOptions.PolicyName);

app.UseHttpsRedirection();

app.UseExceptionHandler();

app.UseResponseCaching();

app.UseRateLimiter();

app.MapHealthChecks("/health");

app.UseStatusCodePages();

await app.RunAsync();
