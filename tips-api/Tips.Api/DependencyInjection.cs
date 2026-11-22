using Asp.Versioning;
using FluentValidation;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.Infrastructure;
using System.Text.Json.Serialization;
using System.Threading.RateLimiting;
using Tips.Api.Middleware;
using Tips.Api.Models;
using Tips.Api.Realtime;
using Tips.Api.Repositories;
using Tips.Api.Repositories.Interfaces;
using Tips.Api.Settings;

namespace Tips.Api;

public static class DependencyInjection
{
    internal const string SubmitRateLimitPolicy = "submit";

    public static WebApplicationBuilder AddApplicationServices(this WebApplicationBuilder builder)
    {
        builder.Services.AddValidatorsFromAssemblyContaining<Program>(includeInternalTypes: true);

        // Enums go over the wire as names ("EntryAdded", "Beginner"), which is
        // what the Go client expects and what query parameters already accept.
        builder.Services.ConfigureHttpJsonOptions(options =>
            options.SerializerOptions.Converters.Add(new JsonStringEnumConverter()));

        builder.Services.AddOutputCache();
        builder.Services.AddHealthChecks();
        builder.Services.AddMetrics();
        builder.Services.AddSingleton(TimeProvider.System);

        builder.Services.AddSingleton<ITipRepository, InMemoryTipRepository>();
        builder.Services.AddSingleton<ILeaderboardRepository, InMemoryLeaderboardRepository>();

        builder.Services.AddSingleton(new SseBroadcasterOptions(
            Name: "Tips", EventType: "tip", ChannelCapacity: 20, ReplayBufferSize: 50));
        builder.Services.AddSingleton(new SseBroadcasterOptions(
            Name: "Leaderboard", EventType: "leaderboard-change", ChannelCapacity: 50, ReplayBufferSize: 100));
        builder.Services.AddSingleton(sp => CreateBroadcaster<GameTip>(sp, "Tips"));
        builder.Services.AddSingleton(sp => CreateBroadcaster<LeaderboardChangeEvent>(sp, "Leaderboard"));

        builder.Services.AddApiVersioning(options =>
        {
            options.DefaultApiVersion = new ApiVersion(1);
            options.ApiVersionReader = new UrlSegmentApiVersionReader();
        }).AddApiExplorer(options =>
        {
            options.GroupNameFormat = "'v'V";
            options.SubstituteApiVersionInUrl = true;
        }).AddOpenApi();

        return builder;
    }

    public static WebApplicationBuilder AddErrorHandling(this WebApplicationBuilder builder)
    {
        builder.Services.AddProblemDetails(options =>
        {
            options.CustomizeProblemDetails = context =>
            {
                context.ProblemDetails.Extensions.TryAdd("requestId", context.HttpContext.TraceIdentifier);
            };
        });
        builder.Services.AddExceptionHandler<ValidationExceptionHandler>();
        builder.Services.AddExceptionHandler<GlobalExceptionHandler>();

        return builder;
    }

    public static WebApplicationBuilder AddBackgroundJobs(this WebApplicationBuilder builder)
    {
        builder.Services.AddHostedService<TipsProducerService>();

        return builder;
    }

    public static WebApplicationBuilder AddCorsPolicy(this WebApplicationBuilder builder)
    {
        string[] allowedOrigins = builder.Configuration
            .GetSection(CorsOptions.SectionName)
            .Get<CorsOptions>()?
            .AllowedOrigins
            .Where(origin => !string.IsNullOrWhiteSpace(origin))
            .ToArray() ?? [];

        builder.Services.AddCors(options =>
        {
            options.AddPolicy(CorsOptions.PolicyName, policy =>
            {
                // With no configured origins the policy allows nothing cross-origin.
                policy
                    .WithOrigins(allowedOrigins)
                    .AllowAnyMethod()
                    .AllowAnyHeader();
            });
        });

        return builder;
    }

    public static WebApplicationBuilder AddRateLimiting(this WebApplicationBuilder builder)
    {
        builder.Services.AddRateLimiter(options =>
        {
            options.RejectionStatusCode = StatusCodes.Status429TooManyRequests;
            options.OnRejected = async (context, token) =>
            {
                string detail = "Too many requests. Please try again later.";
                if (context.Lease.TryGetMetadata(MetadataName.RetryAfter, out TimeSpan retryAfter))
                {
                    context.HttpContext.Response.Headers.RetryAfter = $"{(int)Math.Ceiling(retryAfter.TotalSeconds)}";
                    detail = $"Too many requests. Please try again after {retryAfter.TotalSeconds:0} seconds.";
                }

                ProblemDetailsFactory problemDetailsFactory =
                    context.HttpContext.RequestServices.GetRequiredService<ProblemDetailsFactory>();
                ProblemDetails problemDetails = problemDetailsFactory.CreateProblemDetails(
                    context.HttpContext,
                    StatusCodes.Status429TooManyRequests,
                    "Too Many Requests",
                    detail: detail);
                await context.HttpContext.Response.WriteAsJsonAsync(problemDetails, token);
            };

            // Partition per client IP so one noisy client cannot lock everybody out.
            options.AddPolicy(SubmitRateLimitPolicy, context =>
                RateLimitPartition.GetFixedWindowLimiter(
                    context.Connection.RemoteIpAddress?.ToString() ?? "unknown",
                    _ => new FixedWindowRateLimiterOptions
                    {
                        PermitLimit = 5,
                        Window = TimeSpan.FromMinutes(1),
                    }));
        });

        return builder;
    }

    private static SseBroadcaster<T> CreateBroadcaster<T>(IServiceProvider sp, string name) =>
        ActivatorUtilities.CreateInstance<SseBroadcaster<T>>(
            sp,
            sp.GetServices<SseBroadcasterOptions>().Single(o => o.Name == name));
}
