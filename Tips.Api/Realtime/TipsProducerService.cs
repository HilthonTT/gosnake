using System.Diagnostics;
using System.Diagnostics.Metrics;
using Tips.Api.Models;
using Tips.Api.Repositories.Interfaces;

namespace Tips.Api.Realtime;

internal sealed class TipsProducerService(
    BroadcastManager broadcastManager,
    ITipRepository tipRepository,
    ILogger<TipsProducerService> logger,
    IMeterFactory meterFactory) : BackgroundService
{
    private static readonly TimeSpan TipInterval = TimeSpan.FromSeconds(8);

    private readonly Counter<long> _tipsbroadcastCounter =
        meterFactory.Create("SnakeTips.Realtime").CreateCounter<long>(
            "snaketips.tips_broadcast",
            description: "Total number of tip broadcasts");

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        logger.LogInformation("Tips producer started");

        var allTips = tipRepository.GetAll().ToList();
        var queue = new Queue<GameTip>(Shuffle(allTips));

        while (!stoppingToken.IsCancellationRequested)
        {
            // Refill and reshuffle when we exhaust the deck.
            if (queue.Count == 0)
            {
                queue = new Queue<GameTip>(Shuffle(allTips));
            }

            var tip = queue.Dequeue();

            // Skip broadcast if nobody is listening — no point buffering.
            if (broadcastManager.SubscriberCount > 0)
            {
                var delivered = broadcastManager.Broadcast(tip);
                _tipsbroadcastCounter.Add(1,
                    new TagList { { "category", tip.Category.ToString() } });

                if (logger.IsEnabled(LogLevel.Debug))
                {
                    logger.LogDebug(
                        "Broadcast {TipId} ({Category}/{Difficulty}) to {Count} subscriber(s)",
                        tip.TipId, tip.Category, tip.Difficulty, delivered);
                }
            }
            else
            {
                if (logger.IsEnabled(LogLevel.Debug))
                {
                    logger.LogDebug("No subscribers — skipping tip {TipId}", tip.TipId);
                }
            }

            try
            {
                await Task.Delay(TipInterval, stoppingToken);
            }
            catch (OperationCanceledException)
            {
                break;
            }
        }

        logger.LogInformation("Tips producer stopped");
    }

    private static List<T> Shuffle<T>(IList<T> source)
    {
        var copy = source.ToList();
        var rng = Random.Shared;
        for (var i = copy.Count - 1; i > 0; i--)
        {
            var j = rng.Next(i + 1);
            (copy[i], copy[j]) = (copy[j], copy[i]);
        }
        return copy;
    }
}
