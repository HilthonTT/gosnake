using Microsoft.AspNetCore.Http.HttpResults;
using Microsoft.AspNetCore.Mvc;
using Tips.Api.DTOs.Common;
using Tips.Api.Models;
using Tips.Api.Realtime;
using Tips.Api.Repositories.Interfaces;

namespace Tips.Api.Endpoints;

internal static class TipsStream
{
    public static IEndpointRouteBuilder MapTipsEndpoints(this IEndpointRouteBuilder routes)
    {
        RouteGroupBuilder group = routes.MapGroup("tips")
            .WithTags("Tips");

        group.MapGet("/", GetTips)
            .WithName("GetTips");

        group.MapGet("realtime", StreamTips)
            .WithName("TipsRealtime");

        return routes;
    }

    private static Ok<CollectionResponse<GameTip>> GetTips(
        ITipRepository repo,
        [FromQuery] TipDifficulty? difficulty,
        [FromQuery] TipCategory? category)
    {
        var tips = repo.GetAll().Where(tip => Matches(tip, difficulty, category));
        return TypedResults.Ok(new CollectionResponse<GameTip> { Items = [.. tips] });
    }

    private static ServerSentEventsResult<GameTip> StreamTips(
        SseBroadcaster<GameTip> broadcaster,
        [FromQuery] TipDifficulty? difficulty,
        [FromQuery] TipCategory? category,
        [FromHeader(Name = "Last-Event-ID")] string? lastEventId,
        CancellationToken cancellationToken)
    {
        var stream = broadcaster.StreamAsync(
            lastEventId,
            tip => Matches(tip, difficulty, category),
            cancellationToken);

        return TypedResults.ServerSentEvents(stream);
    }

    private static bool Matches(GameTip? tip, TipDifficulty? difficulty, TipCategory? category) =>
        tip is not null
        && (!difficulty.HasValue || tip.Difficulty == difficulty.Value)
        && (!category.HasValue || tip.Category == category.Value);
}
