using Microsoft.AspNetCore.Mvc;
using System.Net.ServerSentEvents;
using Tips.Api.DTOs.Common;
using Tips.Api.Models;
using Tips.Api.Realtime;
using Tips.Api.Repositories.Interfaces;

namespace Tips.Api.Endpoints;

internal static class TipsStream
{
    private static readonly TimeSpan SseRetryInterval = TimeSpan.FromSeconds(5);

    public static void MapTipsEndpoints(this WebApplication _, IEndpointRouteBuilder routes)
    {
        RouteGroupBuilder group = routes.MapGroup("tips")
            .WithTags("Tips");

        group.MapGet("/", (
            ITipRepository repo,
            [FromQuery] TipDifficulty? difficulty,
            [FromQuery] TipCategory? category) =>
        {
            IReadOnlyList<GameTip> tips = difficulty.HasValue ? repo.GetByDifficulty(difficulty.Value)
                     : category.HasValue ? repo.GetByCategory(category.Value)
                     : repo.GetAll();

            var response = new CollectionResponse<GameTip>()
            {
                Items = tips.ToList()
            };

            return Results.Ok(response);
        })
        .WithName("GetTips");

        group.MapGet("/realtime", (
            BroadcastManager broadcastManager,
            TipEventBuffer eventBuffer,
            ILoggerFactory loggerFactory,
            [FromQuery] TipDifficulty? difficulty,
            [FromQuery] TipCategory? category,
            [FromHeader(Name = "Last-Event-ID")] string? lastEventId,
            CancellationToken cancellationToken) =>
        {
            ILogger logger = loggerFactory.CreateLogger("SnakeTips.Realtime.Stream");
            (string? connectionId, System.Threading.Channels.ChannelReader<GameTip>? reader) = 
                broadcastManager.Subscribe();

            async IAsyncEnumerable<SseItem<GameTip>> StreamTips()
            {
                yield return new SseItem<GameTip>(default!)
                {
                    ReconnectionInterval  = SseRetryInterval,
                };

                if (!string.IsNullOrWhiteSpace(lastEventId))
                {
                    var missed = eventBuffer.GetEventsAfter(lastEventId);

                    if (logger.IsEnabled(LogLevel.Debug))
                    {
                        logger.LogDebug(
                            "Replaying {Count} missed tips for {ConnectionId}",
                            missed.Count, connectionId);
                    }

                    foreach (var item in missed.Where(i => Matches(i.Data, difficulty, category)))
                    {
                        yield return item;
                    }
                }

                await foreach (GameTip tip in reader.ReadAllAsync(cancellationToken))
                {
                    if (!Matches(tip, difficulty, category))
                    {
                        continue;
                    }

                    SseItem<GameTip> sseItem = eventBuffer.Add(tip);
                    yield return sseItem;
                }
            }

            return TypedResults.ServerSentEvents(StreamTips(), eventType: "tip");
        })
        .WithName("TipsRealtime");
    }

    private static bool Matches(GameTip? tip, TipDifficulty? difficulty, TipCategory? category)
    {
        if (tip is null)
        {
            return false;
        }
        if (difficulty.HasValue && tip.Difficulty != difficulty.Value)
        {
            return false;
        }
        if (category.HasValue && tip.Category != category.Value)
        {
            return false;
        }
        return true;
    }
}
