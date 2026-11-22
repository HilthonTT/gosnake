using FluentValidation;
using Microsoft.AspNetCore.Http.HttpResults;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.OutputCaching;
using Tips.Api.DTOs.Common;
using Tips.Api.DTOs.Leaderboard;
using Tips.Api.Models;
using Tips.Api.Realtime;
using Tips.Api.Repositories.Interfaces;

namespace Tips.Api.Endpoints;

internal static class LeaderboardEndpoints
{
    internal const string CacheTag = "leaderboard";

    public static IEndpointRouteBuilder MapLeaderboardEndpoints(this IEndpointRouteBuilder routes)
    {
        RouteGroupBuilder group = routes.MapGroup("leaderboard")
            .WithTags("Leaderboard");

        group.MapGet("/", GetLeaderboard)
            .WithName("GetLeaderboard")
            .CacheOutput(p => p.Expire(TimeSpan.FromSeconds(10)).SetVaryByQuery("top").Tag(CacheTag));

        group.MapGet("realtime", StreamLeaderboard)
            .WithName("LeaderboardRealtime");

        group.MapGet("player/{playerName}", GetByPlayer)
            .WithName("GetPlayerScores");

        group.MapGet("{entryId}", GetById)
            .WithName("GetLeaderboardEntry");

        group.MapPost("/", SubmitScore)
            .WithName("SubmitScore")
            .RequireRateLimiting(DependencyInjection.SubmitRateLimitPolicy);

        group.MapDelete("{entryId}", DeleteEntry)
            .WithName("DeleteLeaderboardEntry");

        return routes;
    }

    private static Ok<CollectionResponse<LeaderboardEntry>> GetLeaderboard(
        ILeaderboardRepository repo,
        [FromQuery] int top = 100)
    {
        top = Math.Clamp(top, 1, 1000);
        return TypedResults.Ok(new CollectionResponse<LeaderboardEntry> { Items = [.. repo.GetTopN(top)] });
    }

    private static Results<Ok<CollectionResponse<LeaderboardEntry>>, NotFound> GetByPlayer(
        string playerName,
        ILeaderboardRepository repo)
    {
        var entries = repo.GetByPlayer(playerName);

        return entries.Count > 0
            ? TypedResults.Ok(new CollectionResponse<LeaderboardEntry> { Items = [.. entries] })
            : TypedResults.NotFound();
    }

    private static Results<Ok<LeaderboardEntry>, NotFound> GetById(string entryId, ILeaderboardRepository repo)
    {
        return repo.GetById(entryId) is { } entry
            ? TypedResults.Ok(entry)
            : TypedResults.NotFound();
    }

    private static async Task<Created<LeaderboardEntry>> SubmitScore(
        SubmitScoreRequest request,
        ILeaderboardRepository repo,
        IValidator<SubmitScoreRequest> validator,
        SseBroadcaster<LeaderboardChangeEvent> broadcaster,
        IOutputCacheStore cache,
        CancellationToken cancellationToken)
    {
        // Throws ValidationException, which ValidationExceptionHandler maps to a 400.
        await validator.ValidateAndThrowAsync(request, cancellationToken);

        var entry = repo.Add(
            request.PlayerName.Trim(),
            request.Score,
            request.Level,
            request.SnakeLength);

        broadcaster.Publish(new LeaderboardChangeEvent(LeaderboardChangeType.EntryAdded, entry));
        await cache.EvictByTagAsync(CacheTag, cancellationToken);

        return TypedResults.Created($"/api/v1/leaderboard/{entry.EntryId}", entry);
    }

    private static async Task<Results<NoContent, NotFound>> DeleteEntry(
        string entryId,
        ILeaderboardRepository repo,
        SseBroadcaster<LeaderboardChangeEvent> broadcaster,
        IOutputCacheStore cache,
        CancellationToken cancellationToken)
    {
        if (repo.Delete(entryId) is not { } removed)
        {
            return TypedResults.NotFound();
        }

        broadcaster.Publish(new LeaderboardChangeEvent(LeaderboardChangeType.EntryDeleted, removed));
        await cache.EvictByTagAsync(CacheTag, cancellationToken);

        return TypedResults.NoContent();
    }

    private static ServerSentEventsResult<LeaderboardChangeEvent> StreamLeaderboard(
        SseBroadcaster<LeaderboardChangeEvent> broadcaster,
        [FromHeader(Name = "Last-Event-ID")] string? lastEventId,
        CancellationToken cancellationToken)
    {
        return TypedResults.ServerSentEvents(broadcaster.StreamAsync(lastEventId, filter: null, cancellationToken));
    }
}
