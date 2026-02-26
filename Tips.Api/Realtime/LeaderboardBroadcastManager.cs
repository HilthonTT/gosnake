using System.Collections.Concurrent;
using System.Diagnostics.Metrics;
using System.Threading.Channels;
using Tips.Api.Models;

namespace Tips.Api.Realtime;

internal sealed class LeaderboardBroadcastManager : IDisposable
{
    private const int ChannelCapacity = 50;

    private readonly ConcurrentDictionary<string, Channel<LeaderboardChangeEvent>> _subscribers = new();
    private readonly ILogger<LeaderboardBroadcastManager> _logger;
    private readonly Meter _meter;
    private readonly ObservableGauge<int> _subscriberGauge;

    public LeaderboardBroadcastManager(
        ILogger<LeaderboardBroadcastManager> logger,
        IMeterFactory meterFactory)
    {
        _logger = logger;
        _meter = meterFactory.Create("SnakeTips.Leaderboard");
        _subscriberGauge = _meter.CreateObservableGauge(
            "snaketips.leaderboard_subscribers",
            () => _subscribers.Count,
            description: "Number of active leaderboard SSE subscribers");
    }

    /// <summary>
    /// Registers a new subscriber and returns a channel reader to stream from.
    /// </summary>
    public (string connectionId, ChannelReader<LeaderboardChangeEvent> reader) Subscribe()
    {
        var connectionId = Guid.NewGuid().ToString("N");
        var channel = Channel.CreateBounded<LeaderboardChangeEvent>(
            new BoundedChannelOptions(ChannelCapacity)
            {
                FullMode = BoundedChannelFullMode.DropOldest,
                SingleReader = true,
                SingleWriter = false,
            });

        _subscribers[connectionId] = channel;

        if (_logger.IsEnabled(LogLevel.Information))
        {
            _logger.LogInformation(
                "Leaderboard subscriber {ConnectionId} connected. Total: {Count}",
                connectionId, _subscribers.Count);
        }

        return (connectionId, channel.Reader);
    }

    /// <summary>Removes and completes the channel when the client disconnects.</summary>
    public void Unsubscribe(string connectionId)
    {
        if (_subscribers.TryRemove(connectionId, out var ch))
        {
            ch.Writer.TryComplete();

            if (_logger.IsEnabled(LogLevel.Information))
            {
                _logger.LogInformation(
                    "Leaderboard subscriber {ConnectionId} disconnected. Total: {Count}",
                    connectionId, _subscribers.Count);
            } 
        }
    }

    /// <summary>
    /// Writes <paramref name="change"/> to every subscriber's channel.
    /// Never blocks — slow clients silently drop their oldest buffered event.
    /// Returns the number of channels that accepted the write.
    /// </summary>
    public int Broadcast(LeaderboardChangeEvent change)
    {
        int delivered = 0;
        foreach (var (id, ch) in _subscribers)
        {
            if (ch.Writer.TryWrite(change))
            {
                delivered++;
            }
            else
            {
                if (_logger.IsEnabled(LogLevel.Debug))
                {
                    _logger.LogDebug(
                        "Leaderboard channel full for {ConnectionId} — oldest event dropped", id);
                }
            }
        }
        return delivered;
    }

    public int SubscriberCount => _subscribers.Count;

    public void Dispose() => _meter.Dispose();
}