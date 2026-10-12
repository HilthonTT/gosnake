using System.Collections.Concurrent;
using System.Diagnostics.Metrics;
using System.Net.ServerSentEvents;
using System.Threading.Channels;

namespace Tips.Api.Realtime;

/// <summary>
/// Fans out events of type <typeparamref name="T"/> to every connected SSE
/// subscriber and keeps a bounded replay buffer so reconnecting clients can
/// resume from their <c>Last-Event-ID</c>.
///
/// Each published event is assigned exactly one ID, regardless of how many
/// subscribers are connected, so IDs are consistent across clients.
/// </summary>
internal sealed class SseBroadcaster<T> : IDisposable
{
    private readonly ConcurrentDictionary<string, Channel<SseItem<T>>> _subscribers = new();
    private readonly LinkedList<SseItem<T>> _buffer = new();
    private readonly Lock _lock = new();
    private readonly SseBroadcasterOptions _options;
    private readonly ILogger _logger;
    private readonly Meter _meter;
    private readonly Counter<long> _publishedCounter;
    private long _nextEventId;

    public SseBroadcaster(SseBroadcasterOptions options, ILoggerFactory loggerFactory, IMeterFactory meterFactory)
    {
        _options = options;
        _logger = loggerFactory.CreateLogger($"SnakeTips.Realtime.{options.Name}");
        _meter = meterFactory.Create($"SnakeTips.Realtime.{options.Name}");
        _meter.CreateObservableGauge(
            $"snaketips.{options.Name.ToLowerInvariant()}.subscribers",
            () => _subscribers.Count,
            description: $"Number of active {options.Name} SSE subscribers");
        _publishedCounter = _meter.CreateCounter<long>(
            $"snaketips.{options.Name.ToLowerInvariant()}.published",
            description: $"Total number of {options.Name} events published");
    }

    public int SubscriberCount => _subscribers.Count;

    /// <summary>
    /// Registers a subscriber. Events the client missed since
    /// <paramref name="lastEventId"/> are queued ahead of live events, so there
    /// are no gaps or duplicates between the replay and the live stream.
    /// </summary>
    public Subscription Subscribe(string? lastEventId)
    {
        var connectionId = Guid.NewGuid().ToString("N");
        var channel = Channel.CreateBounded<SseItem<T>>(new BoundedChannelOptions(_options.ChannelCapacity)
        {
            // A slow client drops old events rather than stalling the publisher.
            FullMode = BoundedChannelFullMode.DropOldest,
            SingleReader = true,
            SingleWriter = false,
        });

        int replayed = 0;
        lock (_lock)
        {
            if (long.TryParse(lastEventId, out var lastId))
            {
                foreach (var item in _buffer)
                {
                    if (long.Parse(item.EventId!) > lastId && channel.Writer.TryWrite(item))
                    {
                        replayed++;
                    }
                }
            }

            _subscribers[connectionId] = channel;
        }

        if (_logger.IsEnabled(LogLevel.Information))
        {
            _logger.LogInformation(
                "Subscriber {ConnectionId} connected ({Replayed} replayed). Total: {Count}",
                connectionId, replayed, _subscribers.Count);
        }

        return new Subscription(this, connectionId, channel.Reader);
    }

    /// <summary>
    /// Assigns the next event ID to <paramref name="data"/>, stores it in the
    /// replay buffer and writes it to every subscriber. Never blocks.
    /// </summary>
    public SseItem<T> Publish(T data)
    {
        SseItem<T> item;
        lock (_lock)
        {
            item = new SseItem<T>(data, _options.EventType)
            {
                EventId = (_nextEventId++).ToString(),
                ReconnectionInterval = _options.ReconnectionInterval,
            };

            _buffer.AddLast(item);
            while (_buffer.Count > _options.ReplayBufferSize)
            {
                _buffer.RemoveFirst();
            }

            foreach (var channel in _subscribers.Values)
            {
                channel.Writer.TryWrite(item);
            }
        }

        _publishedCounter.Add(1);
        return item;
    }

    private void Unsubscribe(string connectionId)
    {
        if (!_subscribers.TryRemove(connectionId, out var channel))
        {
            return;
        }

        channel.Writer.TryComplete();

        if (_logger.IsEnabled(LogLevel.Information))
        {
            _logger.LogInformation(
                "Subscriber {ConnectionId} disconnected. Total: {Count}",
                connectionId, _subscribers.Count);
        }
    }

    public void Dispose() => _meter.Dispose();

    /// <summary>A live subscription; dispose it when the client disconnects.</summary>
    public sealed class Subscription(SseBroadcaster<T> owner, string connectionId, ChannelReader<SseItem<T>> reader)
        : IDisposable
    {
        public string ConnectionId { get; } = connectionId;

        public ChannelReader<SseItem<T>> Reader { get; } = reader;

        public void Dispose() => owner.Unsubscribe(ConnectionId);
    }
}

internal sealed record SseBroadcasterOptions(
    string Name,
    string EventType,
    int ChannelCapacity,
    int ReplayBufferSize)
{
    public TimeSpan ReconnectionInterval { get; init; } = TimeSpan.FromSeconds(5);
}
