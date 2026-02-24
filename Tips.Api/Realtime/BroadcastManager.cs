using System.Collections.Concurrent;
using System.Diagnostics.Metrics;
using System.Threading.Channels;
using Tips.Api.Models;

namespace Tips.Api.Realtime;

/// <summary>
/// Manages a dynamic set of per-connection bounded channels.
/// When the producer calls Broadcast, the tip is written to every
/// currently connected subscriber simultaneously.
///
/// This replaces the per-user ConnectionManager from the Orders demo.
/// Tips are public/identical for all clients so we fan-out rather than route.
/// </summary>
internal sealed class BroadcastManager : IDisposable
{
    private const int ChannelCapacity = 20;

    private readonly ConcurrentDictionary<string, Channel<GameTip>> _subscribers = new();
    private readonly ILogger<BroadcastManager> _logger;
    private readonly Meter _meter;
    private readonly ObservableGauge<int> _subscriberGauge;

    public BroadcastManager(ILogger<BroadcastManager> logger, IMeterFactory meterFactory)
    {
        _logger = logger;
        _meter = meterFactory.Create("SnakeTips.Realtime");
        _subscriberGauge = _meter.CreateObservableGauge(
            "snaketips.active_subscribers",
            () => _subscribers.Count,
            description: "Number of active SSE subscribers");
    }

    public (string connectionId, ChannelReader<GameTip> reader) Subscribe()
    {
        var connectionId = Guid.NewGuid().ToString("N");
        var channel = Channel.CreateBounded<GameTip>(new BoundedChannelOptions(ChannelCapacity)
        {
            // A slow client drops old tips rather than stalling the producer.
            FullMode = BoundedChannelFullMode.DropOldest,
            SingleReader = true,
            SingleWriter = false
        });
        _subscribers[connectionId] = channel;

        if (_logger.IsEnabled(LogLevel.Information))
        {
            _logger.LogInformation(
                "Subscriber {ConnectionId} connected. Total: {Count}",
                connectionId, _subscribers.Count);
        }

        return (connectionId, channel.Reader);
    }

    /// <summary>Removes the channel when the client disconnects.</summary>
    public void Unsubscribe(string connectionId)
    {
        if (_subscribers.TryRemove(connectionId, out var ch))
        {
            ch.Writer.TryComplete();

            if (_logger.IsEnabled(LogLevel.Information))
            {
                _logger.LogInformation(
                    "Subscriber {ConnectionId} disconnected. Total: {Count}",
                    connectionId, _subscribers.Count);
            }
        }
    }

    /// <summary>
    /// Writes the tip to every subscriber's channel.
    /// Subscribers whose channels are full silently drop the oldest tip
    /// (BoundedChannelFullMode.DropOldest), so Broadcast never blocks.
    /// </summary>
    public int Broadcast(GameTip tip)
    {
        int delivered = 0;
        foreach (var (id, ch) in _subscribers)
        {
            if (ch.Writer.TryWrite(tip))
            {
                delivered++;
            }
            else
            {
                if (_logger.IsEnabled(LogLevel.Debug))
                {
                    _logger.LogDebug(
                        "Channel full for subscriber {ConnectionId} — oldest tip dropped", id);
                }
            }
        }
        return delivered;
    }

    public int SubscriberCount => _subscribers.Count;

    public void Dispose() => _meter.Dispose();
}
