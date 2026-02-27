using FluentValidation;
using Microsoft.AspNetCore.Http.HttpResults;
using Microsoft.AspNetCore.Mvc;
using System.Net.ServerSentEvents;
using Tips.Api.DTOs.Common;
using Tips.Api.DTOs.Leaderboard;
using Tips.Api.Models;
using Tips.Api.Realtime;
using Tips.Api.Repositories.Interfaces;

namespace Tips.Api.Endpoints;

internal static class LeaderboardEndpoints
{
    private static readonly TimeSpan SseRetryInterval = TimeSpan.FromSeconds(5);

    public static void MapLeaderboardEndpoints(this WebApplication _, IEndpointRouteBuilder routes)
    {
        RouteGroupBuilder group = routes.MapGroup("leaderboard")
            .WithTags("Leaderboard");

        group.MapGet("/", GetLeaderboard)
            .WithName("GetLeaderboard")
            .CacheOutput(p => p.Expire(TimeSpan.FromSeconds(10)));

        group.MapGet("realtime", StreamLeaderboard)
            .WithName("LeaderboardRealtime");

        group.MapGet("player/{playerName}", GetByPlayer)
            .WithName("GetPlayerScores");

        group.MapPost("/", SubmitScore)
            .WithName("SubmitScore")
            .RequireRateLimiting("default");

        group.MapDelete("{entryId}", DeleteEntry)
            .WithName("DeleteLeaderboardEntry");
    }

    private static Ok<CollectionResponse<LeaderboardEntry>> GetLeaderboard(
        ILeaderboardRepository repo,
        [FromQuery] int top = 100)
    {
        top = Math.Clamp(top, 1, 1000);
        var entries = repo.GetTopN(top);

        var response = new CollectionResponse<LeaderboardEntry>()
        {
            Items = entries.ToList()
        };

        return TypedResults.Ok(response);
    }

    private static Results<Ok<CollectionResponse<LeaderboardEntry>>, NotFound> GetByPlayer(
        string playerName,
        ILeaderboardRepository repo)
    {
        var entries = repo.GetByPlayer(playerName);

        var response = new CollectionResponse<LeaderboardEntry>()
        {
            Items = entries.ToList()
        };

        return entries.Count > 0
            ? TypedResults.Ok(response)
            : TypedResults.NotFound();
    }

    private static async Task<Results<Created<LeaderboardEntry>, ValidationProblem>> SubmitScore(
        SubmitScoreRequest request,
        ILeaderboardRepository repo,
        IValidator<SubmitScoreRequest> validator)
    {
        await validator.ValidateAndThrowAsync(request);

        var entry = repo.Add(
            request.PlayerName.Trim(),
            request.Score,
            request.Level,
            request.SnakeLength);

        return TypedResults.Created($"/leaderboard/{entry.EntryId}", entry);
    }

    private static Results<NoContent, NotFound> DeleteEntry(
        string entryId,
        ILeaderboardRepository repo)
    {
        return repo.Delete(entryId)
            ? TypedResults.NoContent()
            : TypedResults.NotFound();
    }

    private static ServerSentEventsResult<SseItem<LeaderboardChangeEvent>> StreamLeaderboard(
        LeaderboardBroadcastManager broadcastManager,
        LeaderboardEventBuffer eventBuffer,
        ILoggerFactory loggerFactory,
        [FromHeader(Name = "Last-Event-ID")] string? lastEventId,
        CancellationToken cancellationToken)
    {
        ILogger logger = loggerFactory.CreateLogger("SnakeTips.Leaderboard.Stream");
        var (connectionId, reader) = broadcastManager.Subscribe();

        async IAsyncEnumerable<SseItem<LeaderboardChangeEvent>> Stream()
        {
            // Tell the client how long to wait before reconnecting.
            yield return new SseItem<LeaderboardChangeEvent>(default!)
            {
                ReconnectionInterval = SseRetryInterval,
            };

            // Replay any events the client missed since its last connection.
            if (!string.IsNullOrWhiteSpace(lastEventId))
            {
                var missed = eventBuffer.GetEventsAfter(lastEventId);

                if (logger.IsEnabled(LogLevel.Debug))
                {
                    logger.LogDebug(
                        "Replaying {Count} missed leaderboard event(s) for {ConnectionId}",
                        missed.Count, connectionId);
                }

                foreach (var item in missed)
                {
                    yield return item;
                }
            }

            // Stream live events until the client disconnects.
            try
            {
                await foreach (var change in reader.ReadAllAsync(cancellationToken))
                {
                    yield return eventBuffer.Add(change);
                }
            }
            finally
            {
                broadcastManager.Unsubscribe(connectionId);

                if (logger.IsEnabled(LogLevel.Information))
                {
                    logger.LogInformation(
                        "Leaderboard SSE stream closed for {ConnectionId}", connectionId);
                }
            }
        }

        return TypedResults.ServerSentEvents(Stream(), eventType: "leaderboard-change");
    }
}
