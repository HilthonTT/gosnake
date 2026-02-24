using System.Collections.Concurrent;
using System.Diagnostics.Metrics;
using System.Threading.Channels;
using Tips.Api.Models;

namespace Tips.Api.Realtime;

/// <summary>
/// Manages per-user bounded channels. Bounded capacity provides backpressure
/// so a slow or absent consumer cannot cause unbounded memory growth.
/// </summary>
internal sealed class ConnectionManager : IDisposable
{
    private const int ChannelCapacity = 50;

    private readonly ConcurrentDictionary<string, Channel<Tip>> _userChannels = new();
    private readonly ILogger<ConnectionManager> _logger;
    private readonly Meter _meter;
    private readonly ObservableGauge<int> _activeChannelsGauge;

    public ConnectionManager(ILogger<ConnectionManager> logger, IMeterFactory meterFactory)
    {
        _logger = logger;
        _meter = meterFactory.Create("Tips.Realtime");
        _activeChannelsGauge = _meter.CreateObservableGauge(
            "tips.realtime.active_channels",
            () => _userChannels.Count,
            description: "Number of active user channels");
    }

    public Channel<Tip> GetOrCreateChannel(string userId)
    {
        return _userChannels.GetOrAdd(userId, _ => CreateBoundedChannel(userId));
    }

    public ChannelReader<Tip>? GetChannelReader(string userId)
    {
        return _userChannels.TryGetValue(userId, out var ch) ? ch.Reader : null;
    }

    public bool TryGetChannelWriter(string userId, out ChannelWriter<Tip>? writer)
    {
        if (_userChannels.TryGetValue(userId, out var ch))
        {
            writer = ch.Writer;
            return true;
        }
        writer = null;
        return false;
    }

    public void RemoveChannel(string userId)
    {
        if (!_userChannels.TryGetValue(userId,out var ch)) return;

        // Complete the writer so any pending ReadAllAsync unblocks cleanly.
        ch.Writer.TryComplete();

        if (_logger.IsEnabled(LogLevel.Debug))
        {
            _logger.LogDebug("Removed channel for user {UserId}", userId);
        }
    }

    public bool HasActiveChannel(string userId) => _userChannels.ContainsKey(userId);

    public void Dispose()
    {
        _meter.Dispose();
    }

    private Channel<Tip> CreateBoundedChannel(string userId)
    {
        if (_logger.IsEnabled(LogLevel.Debug))
        {
            _logger.LogDebug("Creating bounded channel for user {UserId}", userId);
        }

        return Channel.CreateBounded<Tip>(new BoundedChannelOptions(ChannelCapacity)
        {
            // Drop the oldest item rather than blocking the producer.
            FullMode = BoundedChannelFullMode.DropOldest,
            SingleReader = true,
            SingleWriter = false
        });
    }
}
