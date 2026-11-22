using Tips.Api.Models;
using Tips.Api.Repositories.Interfaces;

namespace Tips.Api.Realtime;

/// <summary>Publishes a shuffled tip every <see cref="TipInterval"/> while anyone is listening.</summary>
internal sealed class TipsProducerService(
    SseBroadcaster<GameTip> broadcaster,
    ITipRepository tipRepository,
    ILogger<TipsProducerService> logger) : BackgroundService
{
    private static readonly TimeSpan TipInterval = TimeSpan.FromSeconds(8);

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        logger.LogInformation("Tips producer started");

        var allTips = tipRepository.GetAll();
        var queue = new Queue<GameTip>();

        using var timer = new PeriodicTimer(TipInterval);
        try
        {
            do
            {
                // Skip broadcast if nobody is listening — no point buffering.
                if (broadcaster.SubscriberCount == 0)
                {
                    continue;
                }

                // Refill and reshuffle when we exhaust the deck.
                if (queue.Count == 0)
                {
                    queue = new Queue<GameTip>(Shuffle(allTips));
                }

                var tip = queue.Dequeue();
                broadcaster.Publish(tip);

                if (logger.IsEnabled(LogLevel.Debug))
                {
                    logger.LogDebug(
                        "Broadcast {TipId} ({Category}/{Difficulty}) to {Count} subscriber(s)",
                        tip.TipId, tip.Category, tip.Difficulty, broadcaster.SubscriberCount);
                }
            }
            while (await timer.WaitForNextTickAsync(stoppingToken));
        }
        catch (OperationCanceledException) when (stoppingToken.IsCancellationRequested)
        {
            // Normal shutdown.
        }

        logger.LogInformation("Tips producer stopped");
    }

    private static List<T> Shuffle<T>(IReadOnlyList<T> source)
    {
        var copy = source.ToList();
        Random.Shared.Shuffle(System.Runtime.InteropServices.CollectionsMarshal.AsSpan(copy));
        return copy;
    }
}
