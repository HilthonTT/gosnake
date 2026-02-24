using Tips.Api;
using Tips.Api.Endpoints;
using Tips.Api.Settings;

var builder = WebApplication.CreateBuilder(args);

builder
    .AddApplicationServices()
    .AddBackgroundJobs()
    .AddCorsPolicy()
    .AddRateLimiting();

var app = builder.Build();

if (app.Environment.IsDevelopment())
{
    app.MapOpenApi();
}

app.UseCookiePolicy();

app.UseHttpsRedirection();

app.UseResponseCaching();

app.UseCors(CorsOptions.PolicyName);

app.UseRateLimiter();

app.MapHealthChecks("/health");

app.MapTipsEndpoints();

await app.RunAsync();
